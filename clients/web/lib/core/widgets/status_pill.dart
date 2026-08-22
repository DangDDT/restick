import "package:material_ui/material_ui.dart";
import "package:restick_web/core/theme/app_theme.dart";
import "package:restick_web/l10n/app_localizations.dart";

class StatusPill extends StatelessWidget {
  const StatusPill({required this.status, super.key});

  final String status;

  String _label(AppLocalizations l10n) {
    switch (status) {
      case "draft":
        return l10n.statusDraft;
      case "processing":
        return l10n.statusProcessing;
      case "awaiting_confirm":
        return l10n.statusConfirm;
      case "pending_review":
        return l10n.statusPending;
      case "published":
        return l10n.statusPublished;
      case "rejected":
        return l10n.statusRejected;
      case "revoked":
        return l10n.statusRevoked;
      default:
        return status;
    }
  }

  @override
  Widget build(BuildContext context) {
    final PackStatusColors colors =
        Theme.of(context).extension<PackStatusColors>()!;
    final Color color = colors.forStatus(status);
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: color.withValues(alpha: 0.4)),
      ),
      child: Text(
        _label(l10n),
        style: Theme.of(context).textTheme.labelSmall?.copyWith(color: color),
      ),
    );
  }
}
