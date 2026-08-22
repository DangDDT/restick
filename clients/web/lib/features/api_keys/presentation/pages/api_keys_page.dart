import "package:flutter/services.dart";
import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:intl/intl.dart";
import "package:restick_web/core/error/error_cubit.dart";
import "package:restick_web/features/api_keys/domain/entities/api_key_entity.dart";
import "package:restick_web/features/api_keys/presentation/cubit/api_key_cubit.dart";
import "package:restick_web/features/auth/presentation/cubit/auth_cubit.dart";
import "package:restick_web/l10n/app_localizations.dart";

class ApiKeysPage extends StatefulWidget {
  const ApiKeysPage({super.key});

  @override
  State<ApiKeysPage> createState() => _ApiKeysPageState();
}

class _ApiKeysPageState extends State<ApiKeysPage>
    with SingleTickerProviderStateMixin {
  TabController? _tabController;

  @override
  void initState() {
    super.initState();
    final bool isAdmin = context.read<AuthCubit>().state.isAdmin;
    if (isAdmin) {
      _tabController = TabController(length: 2, vsync: this);
    }
  }

  @override
  void dispose() {
    _tabController?.dispose();
    super.dispose();
  }

  Future<void> _showCreateDialog(BuildContext context) async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final TextEditingController nameCtrl =
        TextEditingController(text: "SDK Key");
    final TextEditingController daysCtrl = TextEditingController();

    final bool? confirmed = await showDialog<bool>(
      context: context,
      builder: (BuildContext ctx) {
        return AlertDialog(
          title: Text(l10n.btnCreateApiKey),
          content: SizedBox(
            width: 400,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              spacing: 16,
              children: [
                TextField(
                  controller: nameCtrl,
                  decoration: InputDecoration(
                    labelText: l10n.labelKeyName,
                    border: const OutlineInputBorder(),
                  ),
                  autofocus: true,
                ),
                TextField(
                  controller: daysCtrl,
                  keyboardType: TextInputType.number,
                  decoration: InputDecoration(
                    labelText: l10n.labelExpiresInDays,
                    hintText: "e.g. 30, 90 (leave empty for never)",
                    border: const OutlineInputBorder(),
                  ),
                ),
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(ctx).pop(false),
              child: Text(l10n.btnCancel),
            ),
            FilledButton(
              onPressed: () => Navigator.of(ctx).pop(true),
              child: Text(l10n.btnCreateApiKey),
            ),
          ],
        );
      },
    );

    if ((confirmed ?? false) && context.mounted) {
      final String name = nameCtrl.text.trim().isEmpty
          ? "API Key"
          : nameCtrl.text.trim();
      final int? days = int.tryParse(daysCtrl.text.trim());
      final bool isAdmin = context.read<AuthCubit>().state.isAdmin;
      final ApiKeyEntity? created =
          await context.read<ApiKeyCubit>().createKey(
                name: name,
                expiresInDays: days,
                isAdmin: isAdmin,
              );
      if (created != null && created.rawKey != null && context.mounted) {
        await _showOneTimeKeyDialog(context, created.rawKey!);
      }
    }
  }

  Future<void> _showRotateDialog(
      BuildContext context, ApiKeyEntity keyItem) async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final TextEditingController daysCtrl = TextEditingController();

    final bool? confirmed = await showDialog<bool>(
      context: context,
      builder: (BuildContext ctx) {
        return AlertDialog(
          title: Text(l10n.confirmRotateKeyTitle),
          content: SizedBox(
            width: 420,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              spacing: 16,
              children: [
                Text(l10n.confirmRotateKeyBody),
                TextField(
                  controller: daysCtrl,
                  keyboardType: TextInputType.number,
                  decoration: InputDecoration(
                    labelText: l10n.labelExpiresInDays,
                    hintText: "e.g. 30, 90 (leave empty to keep)",
                    border: const OutlineInputBorder(),
                  ),
                ),
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(ctx).pop(false),
              child: Text(l10n.btnCancel),
            ),
            FilledButton(
              style: FilledButton.styleFrom(
                backgroundColor: Theme.of(context).colorScheme.error,
              ),
              onPressed: () => Navigator.of(ctx).pop(true),
              child: Text(l10n.btnRotateKey),
            ),
          ],
        );
      },
    );

    if ((confirmed ?? false) && context.mounted) {
      final int? days = int.tryParse(daysCtrl.text.trim());
      final bool isAdmin = context.read<AuthCubit>().state.isAdmin;
      final ApiKeyEntity? rotated =
          await context.read<ApiKeyCubit>().rotateKey(
                id: keyItem.id,
                expiresInDays: days,
                isAdmin: isAdmin,
              );
      if (rotated != null && rotated.rawKey != null && context.mounted) {
        await _showOneTimeKeyDialog(context, rotated.rawKey!);
      }
    }
  }

  Future<void> _showDeleteDialog(
    BuildContext context,
    ApiKeyEntity keyItem, {
    bool fromAdminView = false,
  }) async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final bool? confirmed = await showDialog<bool>(
      context: context,
      builder: (BuildContext ctx) {
        return AlertDialog(
          title: Text(l10n.confirmDeleteKeyTitle),
          content: SizedBox(
            width: 400,
            child: Text(l10n.confirmDeleteKeyBody),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(ctx).pop(false),
              child: Text(l10n.btnCancel),
            ),
            FilledButton(
              style: FilledButton.styleFrom(
                backgroundColor: Theme.of(context).colorScheme.error,
              ),
              onPressed: () => Navigator.of(ctx).pop(true),
              child: Text(l10n.btnDeleteKey),
            ),
          ],
        );
      },
    );

    if ((confirmed ?? false) && context.mounted) {
      final bool isAdmin = context.read<AuthCubit>().state.isAdmin;
      await context.read<ApiKeyCubit>().deleteKey(
            id: keyItem.id,
            isAdmin: isAdmin,
            fromAdminView: fromAdminView,
          );
    }
  }

  Future<void> _showOneTimeKeyDialog(
      BuildContext context, String rawKey) async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final ColorScheme colors = Theme.of(context).colorScheme;

    await showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (BuildContext ctx) {
        return AlertDialog(
          title: Row(
            spacing: 8,
            children: [
              Icon(Icons.vpn_key_rounded, color: colors.primary),
              Expanded(child: Text(l10n.dialogCreatedKeyTitle)),
            ],
          ),
          content: SizedBox(
            width: 480,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              spacing: 16,
              children: [
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: colors.errorContainer.withValues(alpha: 0.5),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: colors.error),
                  ),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    spacing: 8,
                    children: [
                      Icon(Icons.warning_amber_rounded,
                          color: colors.error, size: 20),
                      Expanded(
                        child: Text(
                          l10n.dialogCreatedKeyWarning,
                          style: TextStyle(
                            color: colors.onErrorContainer,
                            fontWeight: FontWeight.w600,
                            fontSize: 13,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
                Container(
                  padding: const EdgeInsets.symmetric(
                      horizontal: 14, vertical: 12),
                  decoration: BoxDecoration(
                    color: colors.surfaceContainerHighest,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: colors.outlineVariant),
                  ),
                  child: SelectableText(
                    rawKey,
                    style: const TextStyle(
                      fontFamily: "monospace",
                      fontWeight: FontWeight.bold,
                      fontSize: 14,
                    ),
                  ),
                ),
                FilledButton.icon(
                  onPressed: () {
                    Clipboard.setData(ClipboardData(text: rawKey));
                    context.read<ErrorCubit>().show(
                          kind: ToastKind.success,
                          title: "Success",
                          body: l10n.copiedToClipboard,
                        );
                  },
                  icon: const Icon(Icons.copy_rounded, size: 18),
                  label: Text(l10n.btnCopyKey),
                ),
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(ctx).pop(),
              child: const Text("Done"),
            ),
          ],
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final AuthState auth = context.watch<AuthCubit>().state;
    final bool isAdmin = auth.isAdmin;

    return BlocConsumer<ApiKeyCubit, ApiKeyState>(
      listener: (BuildContext context, ApiKeyState state) {
        if (state.errorMessage != null) {
          context.read<ErrorCubit>().showFailure(state.errorMessage!);
        }
      },
      builder: (BuildContext context, ApiKeyState state) {
        return Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            spacing: 16,
            children: [
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        l10n.titleApiKeys,
                        style: Theme.of(context).textTheme.headlineSmall,
                      ),
                      const SizedBox(height: 4),
                      Text(
                        l10n.descApiKeys,
                        style: Theme.of(context).textTheme.bodyMedium,
                      ),
                    ],
                  ),
                  FilledButton.icon(
                    onPressed: () => _showCreateDialog(context),
                    icon: const Icon(Icons.add_rounded),
                    label: Text(l10n.btnCreateApiKey),
                  ),
                ],
              ),
              if (isAdmin && _tabController != null)
                TabBar(
                  controller: _tabController,
                  tabs: [
                    Tab(text: l10n.tabPersonalKeys),
                    Tab(text: l10n.tabSystemKeys),
                  ],
                ),
              if (!state.hasLoaded && state.isRefreshing)
                const LinearProgressIndicator(minHeight: 2)
              else
                Expanded(
                  child: isAdmin && _tabController != null
                      ? TabBarView(
                          controller: _tabController,
                          children: [
                            _buildKeyList(
                              context: context,
                              keys: state.personalKeys,
                              l10n: l10n,
                            ),
                            _buildKeyList(
                              context: context,
                              keys: state.allKeys,
                              l10n: l10n,
                              isAdminView: true,
                            ),
                          ],
                        )
                      : _buildKeyList(
                          context: context,
                          keys: state.personalKeys,
                          l10n: l10n,
                        ),
                ),
            ],
          ),
        );
      },
    );
  }

  Widget _buildKeyList({
    required BuildContext context,
    required List<ApiKeyEntity> keys,
    required AppLocalizations l10n,
    bool isAdminView = false,
  }) {
    if (keys.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(32),
          child: Text(
            l10n.emptyApiKeys,
            style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                  color: Theme.of(context).colorScheme.onSurfaceVariant,
                ),
          ),
        ),
      );
    }

    final DateFormat formatter = DateFormat("yyyy-MM-dd HH:mm");
    final ColorScheme colors = Theme.of(context).colorScheme;

    return ListView.separated(
      itemCount: keys.length,
      separatorBuilder: (_, _) => const SizedBox(height: 12),
      itemBuilder: (BuildContext context, int index) {
        final ApiKeyEntity item = keys[index];
        final String createdAtStr =
            formatter.format(item.createdAt.toLocal());
        final String expiresAtStr = item.expiresAt != null
            ? formatter.format(item.expiresAt!.toLocal())
            : l10n.labelExpiresNever;
        final String lastUsedStr = item.lastUsedAt != null
            ? formatter.format(item.lastUsedAt!.toLocal())
            : l10n.labelNeverUsed;

        return Card(
          elevation: 0,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
            side: BorderSide(color: colors.outlineVariant),
          ),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              children: [
                Container(
                  width: 44,
                  height: 44,
                  decoration: BoxDecoration(
                    color: item.isExpired
                        ? colors.errorContainer
                        : colors.primaryContainer,
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Icon(
                    item.isExpired
                        ? Icons.timer_off_outlined
                        : Icons.vpn_key_outlined,
                    color: item.isExpired
                        ? colors.onErrorContainer
                        : colors.onPrimaryContainer,
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    spacing: 4,
                    children: [
                      Row(
                        spacing: 8,
                        children: [
                          Text(
                            item.name,
                            style: Theme.of(context)
                                .textTheme
                                .titleMedium
                                ?.copyWith(
                                  fontWeight: FontWeight.bold,
                                ),
                          ),
                          if (isAdminView && item.userEmail != null)
                            Container(
                              padding: const EdgeInsets.symmetric(
                                  horizontal: 8, vertical: 2),
                              decoration: BoxDecoration(
                                color: colors.secondaryContainer,
                                borderRadius: BorderRadius.circular(6),
                              ),
                              child: Text(
                                item.userEmail!,
                                style: TextStyle(
                                  fontSize: 11,
                                  color: colors.onSecondaryContainer,
                                ),
                              ),
                            ),
                          if (item.isExpired)
                            Container(
                              padding: const EdgeInsets.symmetric(
                                  horizontal: 8, vertical: 2),
                              decoration: BoxDecoration(
                                color: colors.errorContainer,
                                borderRadius: BorderRadius.circular(6),
                              ),
                              child: Text(
                                "Expired",
                                style: TextStyle(
                                  fontSize: 11,
                                  color: colors.onErrorContainer,
                                  fontWeight: FontWeight.bold,
                                ),
                              ),
                            ),
                        ],
                      ),
                      SelectableText(
                        item.keyPrefix.isNotEmpty
                            ? item.keyPrefix
                            : "rstk_live_...",
                        style: TextStyle(
                          fontFamily: "monospace",
                          color: colors.onSurfaceVariant,
                          fontSize: 13,
                        ),
                      ),
                      const SizedBox(height: 2),
                      Wrap(
                        spacing: 16,
                        runSpacing: 4,
                        children: [
                          Text(
                            "${l10n.labelCreatedAt}: $createdAtStr",
                            style: Theme.of(context)
                                .textTheme
                                .bodySmall
                                ?.copyWith(color: colors.onSurfaceVariant),
                          ),
                          Text(
                            "${l10n.labelExpiresAt}: $expiresAtStr",
                            style: Theme.of(context)
                                .textTheme
                                .bodySmall
                                ?.copyWith(color: colors.onSurfaceVariant),
                          ),
                          Text(
                            "${l10n.labelLastUsedAt}: $lastUsedStr",
                            style: Theme.of(context)
                                .textTheme
                                .bodySmall
                                ?.copyWith(color: colors.onSurfaceVariant),
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 16),
                if (!isAdminView)
                  OutlinedButton.icon(
                    onPressed: () => _showRotateDialog(context, item),
                    icon: const Icon(Icons.refresh_rounded, size: 16),
                    label: Text(l10n.btnRotateKey),
                  ),
                const SizedBox(width: 8),
                IconButton(
                  tooltip: l10n.btnDeleteKey,
                  style: IconButton.styleFrom(
                    foregroundColor: colors.error,
                  ),
                  onPressed: () => _showDeleteDialog(
                    context,
                    item,
                    fromAdminView: isAdminView,
                  ),
                  icon: const Icon(Icons.delete_outline_rounded),
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}
