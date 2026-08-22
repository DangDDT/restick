import "dart:io";

import "package:path/path.dart" as p;
import "package:path_provider/path_provider.dart";

class PackFileStore {
  late Directory _root;

  Future<void> open() async {
    final Directory docs = await getApplicationDocumentsDirectory();
    _root = Directory(p.join(docs.path, "restick", "packs"));
    await _root.create(recursive: true);
  }

  Directory get root => _root;

  Directory packDirectory({required String packId, required int version}) {
    return Directory(p.join(root.path, packId, "v$version"));
  }

  Future<void> deleteDirectory(String dirPath) async {
    if (dirPath.isEmpty) {
      return;
    }
    final Directory dir = Directory(dirPath);
    if (await dir.exists()) {
      await dir.delete(recursive: true);
    }
  }
}
