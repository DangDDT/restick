import "dart:convert";
import "dart:io";

import "package:archive/archive.dart";
import "package:crypto/crypto.dart";
import "package:path/path.dart" as p;
import "package:restick/src/data/mappers/json_codec.dart";

class PackDownloader {
  /// Extracts a downloaded bundle after validating every ZIP member and
  /// manifest reference. Nothing is written before the archive is proven safe.
  Future<void> unzipAndVerify({
    required List<int> zipBytes,
    required Directory dest,
    required String expectedChecksum,
  }) async {
    final Archive archive = ZipDecoder().decodeBytes(zipBytes);
    final Map<String, ArchiveFile> members = _validateMembers(zipBytes, archive);
    final ArchiveFile packEntry = members["pack.json"] ??
        (throw const FormatException("missing pack.json"));
    final Map<String, Object?> pack = asJsonMap(
      jsonDecode(utf8.decode(List<int>.from(packEntry.content as Iterable<int>))),
    );
    _validateManifest(pack, members);

    if (dest.existsSync()) {
      await dest.delete(recursive: true);
    }
    await dest.create(recursive: true);
    for (final ArchiveFile entry in archive) {
      if (entry.isSymbolicLink || !entry.isFile) {
        continue;
      }
      final File out = File(p.join(dest.path, entry.name));
      if (!p.isWithin(dest.path, out.path)) {
        throw const FormatException("unsafe archive member");
      }
      await out.parent.create(recursive: true);
      await out.writeAsBytes(
        List<int>.from(entry.content as Iterable<int>),
        flush: true,
      );
    }
    final String actual = await checksumOfPack(dest);
    if (actual != expectedChecksum) {
      await dest.delete(recursive: true);
      throw const FormatException("checksum mismatch");
    }
  }

  Map<String, ArchiveFile> _validateMembers(
    List<int> zipBytes,
    Archive archive,
  ) {
    // The central directory keeps raw names and external attributes, so
    // duplicates and Unix symlinks are rejected before the decoder collapses
    // duplicate names or resolves links.
    final ZipDirectory directory = ZipDirectory.read(InputStream(zipBytes));
    final Set<String> rawNames = <String>{};
    for (final ZipFileHeader header in directory.fileHeaders) {
      final String name = header.filename;
      if (!_isSafePath(name, directory: name.endsWith("/"))) {
        throw FormatException("unsafe archive member: $name");
      }
      if (!rawNames.add(name)) {
        throw FormatException("duplicate archive member: $name");
      }
      final int external = header.externalFileAttributes ?? 0;
      if (header.versionMadeBy >> 8 == 3 && (external >> 16) & 0xF000 == 0xA000) {
        throw FormatException("symbolic link member: $name");
      }
    }
    final Map<String, ArchiveFile> members = <String, ArchiveFile>{};
    for (final ArchiveFile entry in archive) {
      if (entry.isSymbolicLink) {
        throw FormatException("symbolic link member: ${entry.name}");
      }
      if (!entry.isFile) {
        continue;
      }
      if (!_isSafePath(entry.name)) {
        throw FormatException("unsafe archive member: ${entry.name}");
      }
      if (members.containsKey(entry.name)) {
        throw FormatException("duplicate archive member: ${entry.name}");
      }
      members[entry.name] = entry;
    }
    return members;
  }

  void _validateManifest(
    Map<String, Object?> pack,
    Map<String, ArchiveFile> members,
  ) {
    for (final Object? item in asJsonList(pack["stickers"])) {
      final Map<String, Object?> sticker = asJsonMap(item);
      for (final String key in const <String>["file", "thumb"]) {
        final Object? reference = sticker[key];
        if (reference is! String || !_isSafePath(reference)) {
          throw FormatException("unsafe manifest reference: $reference");
        }
        if (!members.containsKey(reference)) {
          throw FormatException("manifest reference missing: $reference");
        }
      }
    }
  }
}

/// Rejects absolute paths, Windows drives/separators, traversal and empty
/// segments. Directory entries may keep one trailing slash.
bool _isSafePath(String value, {bool directory = false}) {
  if (value.isEmpty ||
      value.contains("\u0000") ||
      value.contains("\\") ||
      value.startsWith("/")) {
    return false;
  }
  if (RegExp(r"^[A-Za-z]:").hasMatch(value)) {
    return false;
  }
  final List<String> segments = value.split("/");
  if (directory) {
    if (segments.last.isNotEmpty) {
      return false;
    }
    segments.removeLast();
  }
  if (segments.isEmpty) {
    return false;
  }
  return segments.every(
    (String segment) => segment.isNotEmpty && segment != "." && segment != "..",
  );
}

Future<String> checksumOfPack(Directory dest) async {
  final File packFile = File(p.join(dest.path, "pack.json"));
  final Map<String, Object?> pack = asJsonMap(
    jsonDecode(await packFile.readAsString()),
  );
  final List<int> joined = <int>[];
  for (final Object? item in asJsonList(pack["stickers"])) {
    final Map<String, Object?> sticker = asJsonMap(item);
    final File webp = File(p.join(dest.path, sticker["file"]! as String));
    joined.addAll(await webp.readAsBytes());
  }
  return sha256.convert(joined).toString();
}
