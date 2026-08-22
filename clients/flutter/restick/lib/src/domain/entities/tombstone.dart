import "package:equatable/equatable.dart";

enum TombstoneAction {
  notifyUserAndPurge,
  silentPurge,
}

class Tombstone extends Equatable {
  const Tombstone({
    required this.packId,
    required this.revokedAt,
    required this.reasonCode,
    required this.checksumSha256,
  });

  final String packId;
  final String revokedAt;
  final String reasonCode;
  final String checksumSha256;

  TombstoneAction get action {
    switch (reasonCode) {
      case "dmca":
      case "explicit":
      case "malicious":
        return TombstoneAction.notifyUserAndPurge;
      case "author_request":
      case "tos_breach":
      default:
        return TombstoneAction.silentPurge;
    }
  }

  bool get requiresUserNotification =>
      action == TombstoneAction.notifyUserAndPurge;

  @override
  List<Object?> get props => <Object?>[
        packId,
        revokedAt,
        reasonCode,
        checksumSha256,
      ];
}

class TombstoneSyncResult extends Equatable {
  const TombstoneSyncResult({
    required this.purgedPackIds,
    required this.tombstones,
  });

  final List<String> purgedPackIds;
  final List<Tombstone> tombstones;

  List<Tombstone> get notifiableTombstones => tombstones
      .where((Tombstone tombstone) => tombstone.requiresUserNotification)
      .toList();

  @override
  List<Object?> get props => <Object?>[purgedPackIds, tombstones];
}
