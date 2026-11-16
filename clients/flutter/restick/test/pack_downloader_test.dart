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

List<int> _packJson(List<Map<String, Object?>> stickers) {
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
      _packJson(<Map<String, Object?>>[
        <String, Object?>{"file": "000.webp", "thumb": "thumbs/thumb_000.webp"},
      ]),
    ),
    _ZipEntry("thumb.webp", _thumbBytes),
    _ZipEntry("000.webp", _stickerBytes),
    _ZipEntry("thumbs/thumb_000.webp", _thumbBytes),
    ...extra,
  ]);
}

String _expectedChecksum() => sha256.convert(_stickerBytes).toString();

const List<int> _animatedBytes = <int>[11, 12, 13, 14];
const Map<String, Object?> _animatedManifest = <String, Object?>{
  "file": "animated/000.webp",
  "frames": 2,
  "duration_ms": 167,
  "width": 512,
  "height": 512,
};

/// A bundle whose only sticker declares the animated variant. The expected
/// checksum appends the animation bytes right after the static ones.
List<int> _animatedZip({
  Map<String, Object?>? animated,
  bool includeAnimated = true,
  List<int> animatedBytes = _animatedBytes,
}) {
  return _buildZip(<_ZipEntry>[
    _ZipEntry(
      "pack.json",
      _packJson(<Map<String, Object?>>[
        <String, Object?>{
          "file": "000.webp",
          "thumb": "thumbs/thumb_000.webp",
          "width": 512,
          "height": 512,
          "animated": animated ?? _animatedManifest,
        },
      ]),
    ),
    _ZipEntry("thumb.webp", _thumbBytes),
    _ZipEntry("000.webp", _stickerBytes),
    _ZipEntry("thumbs/thumb_000.webp", _thumbBytes),
    if (includeAnimated) _ZipEntry("animated/000.webp", animatedBytes),
  ]);
}

String _animatedChecksum() =>
    sha256.convert(<int>[..._stickerBytes, ..._animatedBytes]).toString();

const List<int> _stickerBBytes = <int>[21, 22, 23, 24];
const List<int> _animatedBBytes = <int>[31, 32, 33, 34];

/// Two stickers that each carry an animation: the aggregate must interleave
/// static, animation, static, animation — not all statics then all animations.
List<int> _twoStickerAnimatedZip() {
  return _buildZip(<_ZipEntry>[
    _ZipEntry(
      "pack.json",
      _packJson(<Map<String, Object?>>[
        <String, Object?>{
          "file": "000.webp",
          "thumb": "thumbs/thumb_000.webp",
          "width": 512,
          "height": 512,
          "animated": _animatedManifest,
        },
        <String, Object?>{
          "file": "001.webp",
          "thumb": "thumbs/thumb_001.webp",
          "width": 512,
          "height": 512,
          "animated": <String, Object?>{..._animatedManifest, "file": "animated/001.webp"},
        },
      ]),
    ),
    _ZipEntry("thumb.webp", _thumbBytes),
    _ZipEntry("000.webp", _stickerBytes),
    _ZipEntry("001.webp", _stickerBBytes),
    _ZipEntry("animated/000.webp", _animatedBytes),
    _ZipEntry("animated/001.webp", _animatedBBytes),
    _ZipEntry("thumbs/thumb_000.webp", _thumbBytes),
    _ZipEntry("thumbs/thumb_001.webp", _thumbBytes),
  ]);
}

String _interleavedChecksum() => sha256.convert(<int>[
      ..._stickerBytes,
      ..._animatedBytes,
      ..._stickerBBytes,
      ..._animatedBBytes,
    ]).toString();

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
        _ZipEntry(r"thumbs\evil.webp", _stickerBytes),
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
          _packJson(<Map<String, Object?>>[
            <String, Object?>{
              "file": "../evil.webp",
              "thumb": "thumbs/thumb_000.webp",
            },
          ]),
        ),
        _ZipEntry("thumb.webp", _thumbBytes),
        _ZipEntry("000.webp", _stickerBytes),
        _ZipEntry("thumbs/thumb_000.webp", _thumbBytes),
      ]),
    );
  });

  test("rejects animated manifest reference escape", () async {
    await _expectRejected(
      (Directory root) => _animatedZip(
        animated: <String, Object?>{..._animatedManifest, "file": "../evil.webp"},
      ),
    );
  });

  test("rejects animated manifest reference missing from the archive",
      () async {
    await _expectRejected(
      (Directory root) => _animatedZip(includeAnimated: false),
    );
  });

  test("extracts an animated bundle hashed static then animation", () async {
    final Directory root = await Directory.systemTemp.createTemp("restick_zip_");
    addTearDown(() => root.delete(recursive: true));
    final Directory dest = Directory(p.join(root.path, "dest"));

    await PackDownloader().unzipAndVerify(
      zipBytes: _animatedZip(),
      dest: dest,
      expectedChecksum: _animatedChecksum(),
    );

    expect(
      File(p.join(dest.path, "animated", "000.webp")).readAsBytesSync(),
      _animatedBytes,
    );
    expect(File(p.join(dest.path, "000.webp")).readAsBytesSync(), _stickerBytes);
  });

  test("rejects a checksum that omits the animation", () async {
    final Directory root = await Directory.systemTemp.createTemp("restick_zip_");
    addTearDown(() => root.delete(recursive: true));
    final Directory dest = Directory(p.join(root.path, "dest"));

    await expectLater(
      PackDownloader().unzipAndVerify(
        zipBytes: _animatedZip(),
        dest: dest,
        expectedChecksum: _expectedChecksum(),
      ),
      throwsA(isA<FormatException>()),
    );
    expect(dest.existsSync(), isFalse);
  });

  test("rejects corrupt animation bytes", () async {
    final Directory root = await Directory.systemTemp.createTemp("restick_zip_");
    addTearDown(() => root.delete(recursive: true));
    final Directory dest = Directory(p.join(root.path, "dest"));

    await expectLater(
      PackDownloader().unzipAndVerify(
        zipBytes: _animatedZip(animatedBytes: <int>[0, 0, 0, 0]),
        dest: dest,
        expectedChecksum: _animatedChecksum(),
      ),
      throwsA(isA<FormatException>()),
    );
    expect(dest.existsSync(), isFalse);
  });

  test("hashes two animated stickers interleaved static then animation",
      () async {
    final Directory root = await Directory.systemTemp.createTemp("restick_zip_");
    addTearDown(() => root.delete(recursive: true));
    final Directory dest = Directory(p.join(root.path, "dest"));

    await PackDownloader().unzipAndVerify(
      zipBytes: _twoStickerAnimatedZip(),
      dest: dest,
      expectedChecksum: _interleavedChecksum(),
    );

    expect(
      File(p.join(dest.path, "animated", "000.webp")).readAsBytesSync(),
      _animatedBytes,
    );
    expect(
      File(p.join(dest.path, "animated", "001.webp")).readAsBytesSync(),
      _animatedBBytes,
    );
    expect(File(p.join(dest.path, "001.webp")).readAsBytesSync(), _stickerBBytes);
  });

  test("rejects a checksum that groups statics before animations", () async {
    final Directory root = await Directory.systemTemp.createTemp("restick_zip_");
    addTearDown(() => root.delete(recursive: true));
    final Directory dest = Directory(p.join(root.path, "dest"));

    await expectLater(
      PackDownloader().unzipAndVerify(
        zipBytes: _twoStickerAnimatedZip(),
        dest: dest,
        expectedChecksum: sha256.convert(<int>[
          ..._stickerBytes,
          ..._stickerBBytes,
          ..._animatedBytes,
          ..._animatedBBytes,
        ]).toString(),
      ),
      throwsA(isA<FormatException>()),
    );
    expect(dest.existsSync(), isFalse);
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
