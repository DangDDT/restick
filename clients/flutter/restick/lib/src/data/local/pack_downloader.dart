import "dart:convert";
import "dart:io";

import "package:archive/archive.dart";
import "package:crypto/crypto.dart";
import "package:path/path.dart" as p;
import "package:restick/src/data/mappers/json_codec.dart";

class PackDownloader {
  Future<void> unzipAndVerify({
    required List<int> zipBytes,
    required Directory dest,
    required String expectedChecksum,
  }) async {
    if (dest.existsSync()) {
      await dest.delete(recursive: true);
    }
    await dest.create(recursive: true);
    final Archive archive = ZipDecoder().decodeBytes(zipBytes);
    for (final ArchiveFile entry in archive) {
      if (entry.isFile) {
        final File out = File(p.join(dest.path, entry.name));
        await out.parent.create(recursive: true);
        await out.writeAsBytes(
          List<int>.from(entry.content as Iterable<int>),
          flush: true,
        );
      }
    }
    final String actual = await checksumOfPack(dest);
    if (actual != expectedChecksum) {
      await dest.delete(recursive: true);
      throw const FormatException("checksum mismatch");
    }
  }
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
