import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick_web/core/error/error_cubit.dart";
import "package:restick_web/core/error/failure.dart";
import "package:restick_web/core/widgets/authenticated_image.dart";
import "package:restick_web/core/widgets/reason_dialogs.dart";
import "package:restick_web/core/widgets/status_pill.dart";
import "package:restick_web/features/auth/presentation/cubit/auth_cubit.dart";
import "package:restick_web/features/packs/domain/entities/pack.dart";
import "package:restick_web/features/packs/presentation/cubit/pack_detail_cubit.dart";
import "package:restick_web/features/preview/presentation/cubit/preview_cubit.dart";
import "package:restick_web/l10n/app_localizations.dart";

class PackDetailPage extends StatelessWidget {
  const PackDetailPage({super.key});

  Future<void> _confirm(BuildContext context) async {
    final bool? ok = await showDialog<bool>(
      context: context,
      builder: (BuildContext context) => const ConfirmPublishDialog(),
    );
    if (ok != true || !context.mounted) {
      return;
    }
    try {
      await context.read<PackDetailCubit>().confirmPublish();
      if (context.mounted) {
        context.read<ErrorCubit>().show(
              kind: ToastKind.success,
              title: "OK",
              body: AppLocalizations.of(context).btnConfirmPublish,
            );
      }
    } on Failure catch (failure) {
      if (context.mounted) {
        context.read<ErrorCubit>().showFailure(failure.message);
      }
    }
  }

  Future<void> _showEditPackDialog(
      BuildContext context, PackSummary summary) async {
    final bool canEditSlug = <String>[
      "draft",
      "pending_review",
      "awaiting_confirm",
      "rejected",
    ].contains(summary.status);

    final TextEditingController nameCtrl =
        TextEditingController(text: summary.name);
    final TextEditingController slugCtrl =
        TextEditingController(text: summary.slug);
    final TextEditingController tagsCtrl =
        TextEditingController(text: summary.tags.join(", "));

    final bool? saved = await showDialog<bool>(
      context: context,
      builder: (BuildContext dialogCtx) {
        return AlertDialog(
          title: const Text("Chỉnh sửa thông tin bộ Sticker"),
          content: SingleChildScrollView(
            child: SizedBox(
              width: 440,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  TextField(
                    controller: nameCtrl,
                    decoration: const InputDecoration(
                      labelText: "Tên bộ sticker *",
                      hintText: "Nhập tên bộ sticker",
                    ),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: slugCtrl,
                    enabled: canEditSlug,
                    decoration: InputDecoration(
                      labelText: "Slug *",
                      hintText: "slug-bo-sticker",
                      helperText: canEditSlug
                          ? "Chỉ chứa chữ cái thường, số và dấu gạch ngang"
                          : "Slug không thể thay đổi sau khi bộ sticker đã công khai",
                      helperMaxLines: 2,
                    ),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: tagsCtrl,
                    decoration: const InputDecoration(
                      labelText: "Tags (phân cách bởi dấu phẩy)",
                      hintText: "funny, cat, anime",
                    ),
                  ),
                ],
              ),
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(dialogCtx).pop(false),
              child: const Text("Hủy"),
            ),
            FilledButton(
              onPressed: () => Navigator.of(dialogCtx).pop(true),
              child: const Text("Lưu thay đổi"),
            ),
          ],
        );
      },
    );

    if ((saved ?? false) && context.mounted) {
      final String newName = nameCtrl.text.trim();
      final String newSlug = slugCtrl.text.trim();
      final List<String> newTags = tagsCtrl.text
          .split(",")
          .map((String e) => e.trim())
          .where((String e) => e.isNotEmpty)
          .toList();

      if (newName.isEmpty) {
        context
            .read<ErrorCubit>()
            .showFailure("Tên bộ sticker không được để trống");
        return;
      }

      try {
        await context.read<PackDetailCubit>().updatePackInfo(
              name: newName,
              slug: newSlug,
              tags: newTags,
            );
        if (context.mounted) {
          context.read<ErrorCubit>().show(
                kind: ToastKind.success,
                title: "Thành công",
                body: "Đã cập nhật thông tin bộ sticker",
              );
        }
      } on Failure catch (failure) {
        if (context.mounted) {
          context.read<ErrorCubit>().showFailure(failure.message);
        }
      }
    }
  }

  Future<void> _showEditStickerMetadataDialog(
      BuildContext context, StickerPreview sticker) async {
    final TextEditingController emojiCtrl =
        TextEditingController(text: sticker.emojiHint);
    final TextEditingController keywordsCtrl =
        TextEditingController(text: sticker.keywords.join(", "));

    final bool? saved = await showDialog<bool>(
      context: context,
      builder: (BuildContext dialogCtx) {
        final String url = sticker.thumbUrl.isNotEmpty
            ? sticker.thumbUrl
            : sticker.fileUrl;
        return AlertDialog(
          title: Text("Metadata: ${sticker.file}"),
          content: SingleChildScrollView(
            child: SizedBox(
              width: 400,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  SizedBox(
                    height: 120,
                    width: 120,
                    child: AuthenticatedImage(url: url),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: emojiCtrl,
                    decoration: const InputDecoration(
                      labelText: "Emoji gợi ý (Emoji Hint)",
                      hintText: "😀",
                    ),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: keywordsCtrl,
                    decoration: const InputDecoration(
                      labelText: "Từ khóa liên quan (Keywords)",
                      hintText: "cat, happy, funny, meow",
                      helperText: "Phân cách các từ khóa bằng dấu phẩy",
                    ),
                  ),
                ],
              ),
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(dialogCtx).pop(false),
              child: const Text("Hủy"),
            ),
            FilledButton(
              onPressed: () => Navigator.of(dialogCtx).pop(true),
              child: const Text("Lưu Metadata"),
            ),
          ],
        );
      },
    );

    if ((saved ?? false) && context.mounted) {
      final String emojiHint = emojiCtrl.text.trim();
      final List<String> keywords = keywordsCtrl.text
          .split(",")
          .map((String e) => e.trim())
          .where((String e) => e.isNotEmpty)
          .toList();

      try {
        await context.read<PackDetailCubit>().updateStickerMetadata(
              file: sticker.file,
              emojiHint: emojiHint,
              keywords: keywords,
            );
        if (context.mounted) {
          context.read<ErrorCubit>().show(
                kind: ToastKind.success,
                title: "Thành công",
                body: "Đã cập nhật metadata cho sticker ${sticker.file}",
              );
        }
      } on Failure catch (failure) {
        if (context.mounted) {
          context.read<ErrorCubit>().showFailure(failure.message);
        }
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final bool isAdmin = context.watch<AuthCubit>().state.isAdmin;
    return BlocBuilder<PackDetailCubit, PackDetailState>(
      builder: (BuildContext context, PackDetailState state) {
        final PackDetail? detail = state.detail;
        if (detail == null && state.isRefreshing) {
          return const Padding(
            padding: EdgeInsets.all(24),
            child: LinearProgressIndicator(minHeight: 2),
          );
        }
        if (detail == null) {
          return Center(child: Text(state.errorMessage ?? "Not found"));
        }
        return Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            spacing: 16,
            children: [
              Row(
                spacing: 12,
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          children: [
                            Text(
                              detail.summary.name,
                              style: Theme.of(context).textTheme.headlineSmall,
                            ),
                            const SizedBox(width: 8),
                            IconButton(
                              onPressed: () =>
                                  _showEditPackDialog(context, detail.summary),
                              icon: const Icon(Icons.edit_outlined, size: 20),
                              tooltip: "Chỉnh sửa thông tin Pack",
                            ),
                          ],
                        ),
                        if (detail.summary.tags.isNotEmpty)
                          Padding(
                            padding: const EdgeInsets.only(top: 4),
                            child: Wrap(
                              spacing: 6,
                              runSpacing: 4,
                              children: detail.summary.tags.map((String tag) {
                                return Chip(
                                  label: Text("#$tag"),
                                  visualDensity: VisualDensity.compact,
                                  materialTapTargetSize:
                                      MaterialTapTargetSize.shrinkWrap,
                                );
                              }).toList(),
                            ),
                          ),
                      ],
                    ),
                  ),
                  StatusPill(status: detail.summary.status),
                ],
              ),
              Text(
                "ID: ${detail.summary.id} • Slug: ${detail.summary.slug} • Author: ${detail.summary.author}",
                style: Theme.of(context).textTheme.bodySmall,
              ),
              if (detail.rejectReason.isNotEmpty)
                Text(l10n.rejectReasonLabel(detail.rejectReason)),
              if (isAdmin && detail.summary.canConfirm)
                Container(
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(
                    color: Theme.of(context)
                        .colorScheme
                        .primary
                        .withValues(alpha: 0.08),
                    borderRadius: BorderRadius.circular(14),
                    border: Border.all(
                      color: Theme.of(context)
                          .colorScheme
                          .primary
                          .withValues(alpha: 0.35),
                    ),
                  ),
                  child: Row(
                    spacing: 12,
                    children: [
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          spacing: 4,
                          children: [
                            Text(
                              l10n.bannerConfirmTitle,
                              style: Theme.of(context).textTheme.titleSmall,
                            ),
                            Text(
                              l10n.bannerConfirmBody,
                              style: Theme.of(context).textTheme.bodySmall,
                            ),
                          ],
                        ),
                      ),
                      FilledButton.icon(
                        onPressed: () => _confirm(context),
                        icon: const Icon(Icons.publish_rounded),
                        label: Text(l10n.btnConfirmPublish),
                      ),
                    ],
                  ),
                ),
              Row(
                children: [
                  Expanded(
                    child: Text(
                      l10n.sectionStickers,
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                  ),
                  Text(
                    "Bấm icon bút chì để sửa từ khóa (keywords) • ${l10n.hintClickSend}",
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                ],
              ),
              Expanded(
                child: GridView.builder(
                  gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
                    maxCrossAxisExtent: 150,
                    mainAxisSpacing: 10,
                    crossAxisSpacing: 10,
                  ),
                  itemCount: detail.stickers.length,
                  itemBuilder: (BuildContext context, int index) {
                    final StickerPreview sticker = detail.stickers[index];
                    final String url = sticker.thumbUrl.isNotEmpty
                        ? sticker.thumbUrl
                        : sticker.fileUrl;
                    final bool hasKeywords = sticker.keywords.isNotEmpty;

                    return InkWell(
                      onTap: () =>
                          context.read<PreviewCubit>().sendSticker(url),
                      borderRadius: BorderRadius.circular(12),
                      child: Card(
                        child: Stack(
                          children: [
                            Positioned.fill(
                              child: Padding(
                                padding: const EdgeInsets.all(10),
                                child: AuthenticatedImage(url: url),
                              ),
                            ),
                            Positioned(
                              top: 4,
                              right: 4,
                              child: IconButton(
                                icon: Icon(
                                  hasKeywords
                                      ? Icons.label
                                      : Icons.label_outlined,
                                  size: 18,
                                  color: hasKeywords
                                      ? Theme.of(context).colorScheme.primary
                                      : Theme.of(context)
                                          .colorScheme
                                          .outline,
                                ),
                                tooltip: hasKeywords
                                    ? "Keywords: ${sticker.keywords.join(', ')}"
                                    : "Thêm metadata / keywords",
                                onPressed: () => _showEditStickerMetadataDialog(
                                    context, sticker),
                              ),
                            ),
                            Positioned(
                              left: 6,
                              bottom: 6,
                              child: Text(
                                sticker.emojiHint.isNotEmpty
                                    ? sticker.emojiHint
                                    : "",
                                style: const TextStyle(fontSize: 14),
                              ),
                            ),
                            Positioned(
                              right: 6,
                              bottom: 6,
                              child: Text(
                                "#${(index + 1).toString().padLeft(2, "0")}",
                                style: Theme.of(context).textTheme.labelSmall,
                              ),
                            ),
                          ],
                        ),
                      ),
                    );
                  },
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}
