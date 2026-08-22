import "package:file_picker/file_picker.dart";
import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:go_router/go_router.dart";
import "package:restick_web/core/error/error_cubit.dart";
import "package:restick_web/core/error/failure.dart";
import "package:restick_web/features/auth/presentation/cubit/auth_cubit.dart";
import "package:restick_web/features/packs/domain/entities/pack.dart";
import "package:restick_web/features/packs/presentation/cubit/pack_upload_cubit.dart";
import "package:restick_web/l10n/app_localizations.dart";

class NewPackPage extends StatefulWidget {
  const NewPackPage({super.key});

  @override
  State<NewPackPage> createState() => _NewPackPageState();
}

class _NewPackPageState extends State<NewPackPage> {
  final TextEditingController _name = TextEditingController();
  final TextEditingController _slug = TextEditingController();
  final TextEditingController _author = TextEditingController();
  final TextEditingController _tags = TextEditingController();
  String _sourceKind = "sheet";
  List<UploadFile> _files = <UploadFile>[];

  @override
  void initState() {
    super.initState();
    final String? displayName = context.read<AuthCubit>().state.user?.displayName;
    if (displayName != null) {
      _author.text = displayName;
    }
  }

  @override
  void dispose() {
    _name.dispose();
    _slug.dispose();
    _author.dispose();
    _tags.dispose();
    super.dispose();
  }

  Future<void> _pick() async {
    final FilePickerResult? result = await FilePicker.pickFiles(
      allowMultiple: _sourceKind == "files",
      withData: true,
      type: FileType.image,
    );
    if (result == null) {
      return;
    }
    final List<UploadFile> files = <UploadFile>[];
    for (int i = 0; i < result.files.length; i++) {
      final PlatformFile file = result.files[i];
      final List<int>? bytes = file.bytes;
      if (bytes == null) {
        continue;
      }
      final String ext = (file.extension ?? "png").toLowerCase();
      final String name =
          _sourceKind == "sheet" ? "sheet.$ext" : _safeName(file.name, i);
      files.add(
        UploadFile(name: name, bytes: bytes, contentType: _contentType(ext)),
      );
    }
    setState(() {
      _files = files;
    });
  }

  String _safeName(String name, int index) {
    final String cleaned = name.replaceAll(RegExp("[^A-Za-z0-9._-]"), "_");
    if (cleaned.isEmpty) {
      return "file_$index.png";
    }
    return cleaned;
  }

  String _contentType(String ext) {
    if (ext == "jpg" || ext == "jpeg") {
      return "image/jpeg";
    }
    if (ext == "webp") {
      return "image/webp";
    }
    return "image/png";
  }

  Future<void> _submit() async {
    final AppLocalizations l10n = AppLocalizations.of(context);
    if (_files.isEmpty) {
      context.read<ErrorCubit>().showFailure(l10n.chooseImages);
      return;
    }
    final List<String> tags = _tags.text
        .split(",")
        .map((String item) => item.trim())
        .where((String item) => item.isNotEmpty)
        .toList();
    try {
      final PackUploadCubit cubit = context.read<PackUploadCubit>();
      await cubit.submit(
            name: _name.text.trim(),
            slug: _slug.text.trim(),
            author: _author.text.trim(),
            tags: tags,
            sourceKind: _sourceKind,
            files: _files,
          );
      if (!mounted) {
        return;
      }
      final String? id = cubit.state.createdPackId;
      if (id != null) {
        context.go("/packs/$id");
      }
    } on Failure catch (failure) {
      if (mounted) {
        context.read<ErrorCubit>().showFailure(failure.message);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final PackUploadState upload = context.watch<PackUploadCubit>().state;
    final ColorScheme colors = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.all(24),
      child: SingleChildScrollView(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 680),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            spacing: 16,
            children: [
              Text(
                l10n.titleNewPack,
                style: Theme.of(context).textTheme.headlineSmall,
              ),
              Text(
                l10n.descNewPack,
                style: Theme.of(context).textTheme.bodyMedium,
              ),
              Container(
                padding: const EdgeInsets.all(14),
                decoration: BoxDecoration(
                  color: colors.primary.withValues(alpha: 0.08),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: colors.primary.withValues(alpha: 0.3)),
                ),
                child: Text(
                  "${l10n.noticeHeader} ${l10n.noticeBody}",
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
              ),
              TextField(
                controller: _name,
                decoration: InputDecoration(labelText: l10n.labelTitle),
              ),
              TextField(
                controller: _slug,
                decoration: InputDecoration(labelText: l10n.labelSlug),
              ),
              TextField(
                controller: _author,
                decoration: InputDecoration(labelText: l10n.labelAuthor),
              ),
              TextField(
                controller: _tags,
                decoration: InputDecoration(labelText: l10n.labelTags),
              ),
              Text(l10n.labelSource, style: Theme.of(context).textTheme.labelLarge),
              SegmentedButton<String>(
                segments: [
                  ButtonSegment<String>(
                    value: "sheet",
                    label: Text(l10n.sourceSheet),
                  ),
                  ButtonSegment<String>(
                    value: "files",
                    label: Text(l10n.sourceFiles),
                  ),
                ],
                selected: <String>{_sourceKind},
                onSelectionChanged: (Set<String> next) {
                  setState(() {
                    _sourceKind = next.first;
                    _files = <UploadFile>[];
                  });
                },
              ),
              Text(l10n.labelUpload, style: Theme.of(context).textTheme.labelLarge),
              InkWell(
                onTap: _pick,
                borderRadius: BorderRadius.circular(14),
                child: Container(
                  padding: const EdgeInsets.all(28),
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(14),
                    border: Border.all(
                      color: colors.outline,
                    ),
                  ),
                  child: Column(
                    spacing: 8,
                    children: [
                      Icon(Icons.cloud_upload_outlined, color: colors.primary),
                      Text(
                        _files.isEmpty
                            ? l10n.dropzoneText
                            : l10n.filesSelected(_files.length),
                        style: Theme.of(context).textTheme.titleSmall,
                      ),
                      Text(
                        l10n.dropzoneHint,
                        style: Theme.of(context).textTheme.bodySmall,
                      ),
                    ],
                  ),
                ),
              ),
              Row(
                mainAxisAlignment: MainAxisAlignment.end,
                spacing: 12,
                children: [
                  OutlinedButton.icon(
                    onPressed: () => context.go("/packs"),
                    icon: const Icon(Icons.close_rounded),
                    label: Text(l10n.btnCancel),
                  ),
                  FilledButton.icon(
                    onPressed: upload.isSubmitting ? null : _submit,
                    icon: Icon(
                      upload.isSubmitting
                          ? Icons.hourglass_top_rounded
                          : Icons.send_rounded,
                    ),
                    label: Text(
                      upload.isSubmitting ? l10n.uploading : l10n.btnSubmit,
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}
