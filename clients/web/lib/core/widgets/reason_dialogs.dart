import "package:material_ui/material_ui.dart";
import "package:restick_web/features/packs/domain/entities/pack.dart";
import "package:restick_web/l10n/app_localizations.dart";

class ReasonResult {
  const ReasonResult({
    required this.code,
    required this.text,
  });

  final String code;
  final String text;
}

class RejectReasonDialog extends StatefulWidget {
  const RejectReasonDialog({super.key});

  @override
  State<RejectReasonDialog> createState() => _RejectReasonDialogState();
}

class _RejectReasonDialogState extends State<RejectReasonDialog> {
  final TextEditingController _reason = TextEditingController();
  String _code = RejectionCode.copyright.code;

  @override
  void dispose() {
    _reason.dispose();
    super.dispose();
  }

  void _applyPreset(RejectionCode code, bool isVietnamese) {
    setState(() {
      _code = code.code;
      _reason.text = code.defaultText(isVietnamese: isVietnamese);
    });
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final bool isVietnamese =
        Localizations.localeOf(context).languageCode == "vi";
    return AlertDialog(
      title: Text(l10n.modalRejectTitle),
      content: SizedBox(
        width: 480,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          spacing: 12,
          children: [
            Text(l10n.modalRejectBody),
            Text(l10n.labelQuickPresets),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                ActionChip(
                  label: Text(l10n.presetCopyright),
                  onPressed: () =>
                      _applyPreset(RejectionCode.copyright, isVietnamese),
                ),
                ActionChip(
                  label: Text(l10n.presetQuality),
                  onPressed: () =>
                      _applyPreset(RejectionCode.quality, isVietnamese),
                ),
                ActionChip(
                  label: Text(l10n.presetDuplicate),
                  onPressed: () =>
                      _applyPreset(RejectionCode.duplicate, isVietnamese),
                ),
                ActionChip(
                  label: Text(l10n.presetMisleading),
                  onPressed: () =>
                      _applyPreset(RejectionCode.misleading, isVietnamese),
                ),
                ActionChip(
                  label: Text(l10n.presetExplicit),
                  onPressed: () =>
                      _applyPreset(RejectionCode.explicit, isVietnamese),
                ),
              ],
            ),
            TextField(
              controller: _reason,
              maxLines: 4,
              decoration: InputDecoration(labelText: l10n.labelReason),
              onChanged: (_) => setState(() {}),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(l10n.btnCancel),
        ),
        FilledButton(
          onPressed: _reason.text.trim().isEmpty
              ? null
              : () => Navigator.pop(
                    context,
                    ReasonResult(code: _code, text: _reason.text.trim()),
                  ),
          child: Text(l10n.btnSubmitReason),
        ),
      ],
    );
  }
}

class RevokeReasonDialog extends StatefulWidget {
  const RevokeReasonDialog({super.key});

  @override
  State<RevokeReasonDialog> createState() => _RevokeReasonDialogState();
}

class _RevokeReasonDialogState extends State<RevokeReasonDialog> {
  final TextEditingController _reason = TextEditingController();
  String _code = RevocationCode.dmca.code;

  @override
  void dispose() {
    _reason.dispose();
    super.dispose();
  }

  void _applyPreset(RevocationCode code, bool isVietnamese) {
    setState(() {
      _code = code.code;
      _reason.text = code.defaultText(isVietnamese: isVietnamese);
    });
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final bool isVietnamese =
        Localizations.localeOf(context).languageCode == "vi";
    return AlertDialog(
      title: Text(l10n.modalRevokeTitle),
      content: SizedBox(
        width: 480,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          spacing: 12,
          children: [
            Text(l10n.modalRevokeBody),
            Text(l10n.labelQuickPresets),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                ActionChip(
                  label: Text(l10n.presetDmca),
                  onPressed: () =>
                      _applyPreset(RevocationCode.dmca, isVietnamese),
                ),
                ActionChip(
                  label: Text(l10n.presetMalicious),
                  onPressed: () =>
                      _applyPreset(RevocationCode.malicious, isVietnamese),
                ),
                ActionChip(
                  label: Text(l10n.presetAuthorRequest),
                  onPressed: () =>
                      _applyPreset(RevocationCode.authorRequest, isVietnamese),
                ),
                ActionChip(
                  label: Text(l10n.presetTos),
                  onPressed: () =>
                      _applyPreset(RevocationCode.tosBreach, isVietnamese),
                ),
              ],
            ),
            TextField(
              controller: _reason,
              maxLines: 4,
              decoration: InputDecoration(labelText: l10n.labelReason),
              onChanged: (_) => setState(() {}),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(l10n.btnCancel),
        ),
        FilledButton(
          onPressed: _reason.text.trim().isEmpty
              ? null
              : () => Navigator.pop(
                    context,
                    ReasonResult(code: _code, text: _reason.text.trim()),
                  ),
          child: Text(l10n.btnSubmitReason),
        ),
      ],
    );
  }
}

class ConfirmPublishDialog extends StatelessWidget {
  const ConfirmPublishDialog({super.key});

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return AlertDialog(
      title: Text(l10n.publishTitle),
      content: Text(l10n.publishBody),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context, false),
          child: Text(l10n.btnCancel),
        ),
        FilledButton(
          onPressed: () => Navigator.pop(context, true),
          child: Text(l10n.btnPublish),
        ),
      ],
    );
  }
}
