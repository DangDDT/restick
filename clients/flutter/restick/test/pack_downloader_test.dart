import "dart:convert";
import "dart:io";

import "package:archive/archive.dart";
import "package:crypto/crypto.dart";
import "package:flutter_test/flutter_test.dart";
import "package:path/path.dart" as p;
import "package:restick/src/data/local/pack_downloader.dart";

class _ZipEntry {
  _ZipEntry(this.name, this.bytes, {this.unixMode});

  final String name;
  final List<int> bytes;
  final int? unixMode;
}

void _u16(List<int> out, int value) {
  out
    ..add(value & 0xff)
    ..add((value >> 8) & 0xff);
}

void _u32(List<int> out, int value) {
  out
    ..add(value & 0xff)
    ..add((value >> 8) & 0xff)
    ..add((value >> 16) & 0xff)
    ..add((value >> 24) & 0xff);
}

/// Minimal store-only ZIP writer so tests can express raw central-directory
/// names and Unix modes (duplicates, symlinks) that ZipEncoder normalizes.
List<int> _buildZip(List<_ZipEntry> entries) {
  final List<int> out = <int>[];
  final List<int> central = <int>[];
  for (final _ZipEntry entry in entries) {
    final List<int> name = utf8.encode(entry.name);
    final int crc = getCrc32(entry.bytes);
    final int offset = out.length;
    _u32(out, 0x04034b50);
    _u16(out, 20);
    _u16(out, 0);
    _u16(out, 0);
    _u16(out, 0);
    _u16(out, 0);
    _u32(out, crc);
    _u32(out, entry.bytes.length);
    _u32(out, entry.bytes.length);
    _u16(out, name.length);
    _u16(out, 0);
    out
      ..addAll(name)
      ..addAll(entry.bytes);

    _u32(central, 0x02014b50);
    _u16(central, 0x0314);
    _u16(central, 20);
    _u16(central, 0);
    _u16(central, 0);
    _u16(central, 0);
    _u16(central, 0);
    _u32(central, crc);
    _u32(central, entry.bytes.length);
    _u32(central, entry.bytes.length);
    _u16(central, name.length);
    _u16(central, 0);
    _u16(central, 0);
    _u16(central, 0);
    _u16(central, 0);
    _u32(central, (entry.unixMode ?? 0x81A4) << 16);
    _u32(central, offset);
    central.addAll(name);
  }
  final int centralOffset = out.length;
  out.addAll(central);
  _u32(out, 0x06054b50);
  _u16(out, 0);
  _u16(out, 0);
  _u16(out, entries.length);
  _u16(out, entries.length);
  _u32(out, central.length);
  _u32(out, centralOffset);
  _u16(out, 0);
  return out;
}

List<int> _packJson(List<Map<String, String>> stickers) {
  return utf8.encode(
    jsonEncode(<String, Object?>{
      "schema": "restick.pack.v1",
      "id": "pack-id",
      "version": 1,
      "stickers": stickers,
    }),
  );
}

const List<int> _stickerBytes = <int>[1, 2, 3, 4];
const List<int> _thumbBytes = <int>[9, 8, 7];

/// A valid bundle plus extra members. Only the unsafe extra should be able to
/// make extraction fail, so each guard stays discriminating.
List<int> _validZip([List<_ZipEntry> extra = const <_ZipEntry>[]]) {
  return _buildZip(<_ZipEntry>[
    _ZipEntry(
      "pack.json",
      _packJson(<Map<String, String>>[
        <String, String>{"file": "000.webp", "thumb": "thumbs/thumb_000.webp"},
      ]),
    ),
    _ZipEntry("thumb.webp", _thumbBytes),
    _ZipEntry("000.webp", _stickerBytes),
    _ZipEntry("thumbs/thumb_000.webp", _thumbBytes),
    ...extra,
  ]);
}

String _expectedChecksum() => sha256.convert(_stickerBytes).toString();

/// Extracts with a pre-created destination holding a sentinel file and proves
/// that every invalid ZIP is rejected before the destination is deleted or
/// written, and that no file is ever created outside it.
Future<void> _expectRejected(
  List<int> Function(Directory root) buildZip,
) async {
  final Directory root = await Directory.systemTemp.createTemp("restick_zip_");
  addTearDown(() => root.delete(recursive: true));
  final Directory dest = Directory(p.join(root.path, "dest"))
    ..createSync(recursive: true);
  final File sentinel = File(p.join(dest.path, "sentinel.bin"))
    ..writeAsStringSync("keep-me");

  await expectLater(
    PackDownloader().unzipAndVerify(
      zipBytes: buildZip(root),
      dest: dest,
      expectedChecksum: _expectedChecksum(),
    ),
    throwsA(isA<FormatException>()),
  );

  expect(sentinel.existsSync(), isTrue);
  expect(sentinel.readAsStringSync(), "keep-me");
  expect(
    Directory(dest.path)
        .listSync(recursive: true)
        .whereType<File>()
        .map((File file) => p.relative(file.path, from: dest.path)),
    <String>["sentinel.bin"],
  );
  expect(
    Directory(root.path)
        .listSync()
        .map((FileSystemEntity entity) => p.basename(entity.path)),
    <String>["dest"],
    reason: "no file may be written outside the destination",
  );
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test("rejects traversal member", () async {
    await _expectRejected(
      (Directory root) => _validZip(<_ZipEntry>[
        _ZipEntry("../evil.webp", _stickerBytes),
      ]),
    );
  });

  test("rejects absolute member", () async {
    await _expectRejected(
      (Directory root) => _validZip(<_ZipEntry>[
        _ZipEntry(p.join(root.path, "evil.webp"), _stickerBytes),
      ]),
    );
  });

  test("rejects nested escape member", () async {
    await _expectRejected(
      (Directory root) => _validZip(<_ZipEntry>[
        _ZipEntry("thumbs/../../evil.webp", _stickerBytes),
      ]),
    );
  });

  test("rejects windows separator member", () async {
    await _expectRejected(
      (Directory root) => _validZip(<_ZipEntry>[
        _ZipEntry("thumbs\\evil.webp", _stickerBytes),
      ]),
    );
  });

  test("rejects duplicate member", () async {
    await _expectRejected(
      (Directory root) => _validZip(<_ZipEntry>[
        _ZipEntry("extra.webp", <int>[5, 6, 7]),
        _ZipEntry("extra.webp", <int>[8, 9, 10]),
      ]),
    );
  });

  test("rejects symlink member", () async {
    await _expectRejected(
      (Directory root) => _validZip(<_ZipEntry>[
        _ZipEntry("link.webp", utf8.encode("/etc/passwd"), unixMode: 0xA1FF),
      ]),
    );
  });

  test("rejects manifest reference escape", () async {
    // Combined invariant: an unsafe manifest reference must be refused before
    // the destination is mutated; membership would also reject it, but the
    // path guard fires first. No exception message is pinned.
    await _expectRejected(
      (Directory root) => _buildZip(<_ZipEntry>[
        _ZipEntry(
          "pack.json",
          _packJson(<Map<String, String>>[
            <String, String>{"file": "../evil.webp", "thumb": "thumbs/thumb_000.webp"},
          ]),
        ),
        _ZipEntry("thumb.webp", _thumbBytes),
        _ZipEntry("000.webp", _stickerBytes),
        _ZipEntry("thumbs/thumb_000.webp", _thumbBytes),
      ]),
    );
  });

  test("extracts valid nested thumbnails", () async {
    final Directory root = await Directory.systemTemp.createTemp("restick_zip_");
    addTearDown(() => root.delete(recursive: true));
    final Directory dest = Directory(p.join(root.path, "dest"));

    await PackDownloader().unzipAndVerify(
      zipBytes: _validZip(),
      dest: dest,
      expectedChecksum: _expectedChecksum(),
    );

    expect(File(p.join(dest.path, "pack.json")).existsSync(), isTrue);
    expect(File(p.join(dest.path, "000.webp")).readAsBytesSync(), _stickerBytes);
    expect(
      File(p.join(dest.path, "thumbs", "thumb_000.webp")).readAsBytesSync(),
      _thumbBytes,
    );
    expect(
      Directory(root.path)
          .listSync()
          .whereType<File>()
          .map((File file) => p.basename(file.path)),
      isEmpty,
    );
  });

  test("rejects checksum mismatch without writing outside dest", () async {
    final Directory root = await Directory.systemTemp.createTemp("restick_zip_");
    addTearDown(() => root.delete(recursive: true));
    final Directory dest = Directory(p.join(root.path, "dest"));

    await expectLater(
      PackDownloader().unzipAndVerify(
        zipBytes: _validZip(),
        dest: dest,
        expectedChecksum: "0" * 64,
      ),
      throwsA(isA<FormatException>()),
    );
    expect(dest.existsSync(), isFalse);
    expect(
      Directory(root.path)
          .listSync()
          .whereType<File>()
          .map((File file) => p.basename(file.path)),
      isEmpty,
    );
  });
}
