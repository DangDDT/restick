import "dart:async";
import "dart:math" as math;

import "package:flutter_bloc/flutter_bloc.dart";
import "package:material_ui/material_ui.dart";
import "package:restick/src/core/restick_client.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/domain/entities/sticker_preview.dart";
import "package:restick/src/presentation/cubit/marketplace_cubit.dart";
import "package:restick/src/presentation/theme/pack_tones.dart";
import "package:restick/src/presentation/theme/restick_brand.dart";
import "package:restick/src/presentation/theme/restick_strings.dart";
import "package:restick/src/presentation/theme/restick_theme.dart";
import "package:restick/src/presentation/widgets/die_cut_sticker.dart";
import "package:restick/src/presentation/widgets/download_button.dart";
import "package:restick/src/presentation/widgets/restick_mascot.dart";


/// Height of the transparent band above the sheet that the mascot sits in.
const double _perch = 34;
const double _mascotSize = 60;

/// Opens the pack detail sheet (layout D) for [packId] from [cubit].
///
/// The sheet follows the cubit live (download progress, owned, removal). If
/// the pack leaves `cubit.state.packs` (revoked, filtered away by a reload)
/// the sheet closes itself; unknown ids open nothing.
Future<void> showPackDetailSheet(
  BuildContext context, {
  required MarketplaceCubit cubit,
  required String packId,
  RestickStrings strings = RestickStrings.vi,
}) async {
  if (!cubit.state.packs.any((StickerPack p) => p.id == packId)) {
    return;
  }
  final RestickClient? client = context.read<RestickClient?>();
  final ThemeData theme = Theme.of(context);
  await showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    backgroundColor: Colors.transparent,
    elevation: 0,
    // The download burst paints outside the button; only the background
    // layer inside the sheet is rounded.
    clipBehavior: Clip.none,
    barrierColor: RestickTheme.of(context).ink.withValues(alpha: 0.4),
    sheetAnimationStyle:
        MediaQuery.disableAnimationsOf(context)
            ? AnimationStyle.noAnimation
            : const AnimationStyle(
              duration: Duration(milliseconds: 400),
              reverseDuration: Duration(milliseconds: 250),
              curve: RestickMotion.ease,
            ),
    builder: (BuildContext context) {
      final Widget sheet = Theme(
        data: theme,
        child: BlocProvider<MarketplaceCubit>.value(
          value: cubit,
          child: _PackDetailSheet(packId: packId, strings: strings),
        ),
      );
      return client == null ? sheet : RepositoryProvider<RestickClient>.value(value: client, child: sheet);
    },
  );
}

class _PackDetailSheet extends StatefulWidget {
  const _PackDetailSheet({required this.packId, required this.strings});

  final String packId;
  final RestickStrings strings;

  @override
  State<_PackDetailSheet> createState() => _PackDetailSheetState();
}

class _PackDetailSheetState extends State<_PackDetailSheet> {
  /// Last pack seen in the cubit; kept so the closing frame still renders.
  StickerPack? _last;
  bool _closing = false;

  StickerPack? _find(MarketplaceState state) {
    for (final StickerPack pack in state.packs) {
      if (pack.id == widget.packId) {
        return pack;
      }
    }
    return null;
  }

  void _close() {
    final ModalRoute<Object?>? route = ModalRoute.of(context);
    if (_closing || route == null || !route.isActive) {
      return;
    }
    _closing = true;
    if (route.isCurrent) {
      Navigator.of(context).pop();
    } else {
      Navigator.of(context).removeRoute(route);
    }
  }

  @override
  Widget build(BuildContext context) {
    return BlocConsumer<MarketplaceCubit, MarketplaceState>(
      listenWhen:
          (MarketplaceState previous, MarketplaceState next) =>
              _find(next) == null,
      listener: (BuildContext context, MarketplaceState state) => _close(),
      builder: (BuildContext context, MarketplaceState state) {
        // The cubit's own instance, unchanged: the button reads identity.
        final StickerPack? pack = _find(state) ?? _last;
        _last = pack;
        if (pack == null) {
          return const SizedBox.shrink();
        }
        return _SheetBody(
          pack: pack,
          strings: widget.strings,
          failureMessage: state.downloads[pack.id]?.message ?? "",
          cubit: context.read<MarketplaceCubit>(),
        );
      },
    );
  }
}

class _SheetBody extends StatelessWidget {
  const _SheetBody({
    required this.pack,
    required this.strings,
    required this.cubit,
    required this.failureMessage,
  });

  final StickerPack pack;
  final RestickStrings strings;
  final MarketplaceCubit cubit;
  final String failureMessage;

  @override
  Widget build(BuildContext context) {
    final bool reduce = MediaQuery.disableAnimationsOf(context);
    final MediaQueryData media = MediaQuery.of(context);
    final PackTones tones = PackTones.of(pack);
    final double maxHeight =
        media.size.height - media.padding.top - _perch - 24;

    final Widget content = Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: <Widget>[
        Center(
          child: Container(
            width: 40,
            height: 5,
            margin: const EdgeInsets.only(bottom: 14),
            decoration: BoxDecoration(
              color: const Color(0xFFE6DCEB),
              borderRadius: BorderRadius.circular(3),
            ),
          ),
        ),
        Padding(
          // Keeps the name clear of the perched mascot.
          padding: const EdgeInsets.only(right: _mascotSize),
          child: Text(
            pack.name,
            maxLines: 3,
            overflow: TextOverflow.ellipsis,
            style: RestickBrand.text(
              size: 22,
              color: RestickTheme.of(context).ink,
              weight: FontWeight.w700,
              height: 1.15,
            ),
          ),
        ),
        const SizedBox(height: 2),
        Text(
          pack.downloadStatus == PackDownloadStatus.failed && failureMessage.isNotEmpty
              ? failureMessage
              : pack.isDownloaded ? strings.readyInCollection
              : strings.packMeta(pack.stickerCount, pack.author),
          style: RestickBrand.text(
            size: 13,
            weight: FontWeight.w500,
            color: RestickTheme.of(context).mutedInk,
          ),
        ),
        const SizedBox(height: 14),
        DecoratedBox(
          decoration: BoxDecoration(
            color: tones.background,
            borderRadius: BorderRadius.circular(24),
          ),
          child: Padding(
            padding: const EdgeInsets.all(14),
            child: _StickerGrid(pack: pack, reduce: reduce),
          ),
        ),
        if (pack.tags.isNotEmpty) ...<Widget>[
          const SizedBox(height: 12),
          Wrap(
            spacing: 6,
            runSpacing: 6,
            children: <Widget>[
              for (final String tag in pack.tags)
                DecoratedBox(
                  decoration: BoxDecoration(
                    color: RestickTheme.of(context).canvas,
                    borderRadius: BorderRadius.circular(99),
                  ),
                  child: Padding(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 12,
                      vertical: 5,
                    ),
                    child: Text(
                      tag,
                      style: RestickBrand.text(size: 12, color: RestickTheme.of(context).mutedInk),
                    ),
                  ),
                ),
            ],
          ),
        ],
        const SizedBox(height: 18),
        PackDownloadButton(
          pack: pack,
          strings: strings,
          onDownload: () => unawaited(cubit.download(pack.id)),
          onRemove: () => unawaited(cubit.delete(pack.id)),
        ),
      ],
    );

    return Stack(
      key: const ValueKey<String>("restick.sheet"),
      clipBehavior: Clip.none,
      children: <Widget>[
        // Only this background layer is rounded; nothing clips the button.
        const Positioned.fill(
          top: _perch,
          child: DecoratedBox(
            decoration: BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.vertical(top: Radius.circular(28)),
            ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.only(top: _perch),
          child: ConstrainedBox(
            constraints: BoxConstraints(maxHeight: math.max(0, maxHeight)),
            // Scrolled content must not paint above the white edge, but the
            // button's burst may still spill sideways and downward.
            child: ClipRect(
              clipper: const _TopEdgeClipper(),
              child: SingleChildScrollView(
                clipBehavior: Clip.none,
                // Room for the success burst (~76px) under the button.
                padding: EdgeInsets.fromLTRB(
                  20,
                  10,
                  20,
                  80 + media.viewPadding.bottom,
                ),
                child: content,
              ),
            ),
          ),
        ),
        Positioned(
          top: 0,
          right: 24,
          child: _PopIn(
            reduce: reduce,
            delay: 0.35,
            child: const RestickMascot(size: _mascotSize, blink: false),
          ),
        ),
      ],
    );
  }
}

/// 3x2 grid of up to six previews, or one big sticker without previews.
class _StickerGrid extends StatelessWidget {
  const _StickerGrid({required this.pack, required this.reduce});

  final StickerPack pack;
  final bool reduce;

  // Small alternating tilts so the grid never reads as rigid squares.
  static const List<double> _tilts = <double>[
    -0.07,
    0.06,
    -0.05,
    0.08,
    -0.06,
    0.05,
  ];

  @override
  Widget build(BuildContext context) {
    final List<String> urls =
        pack.previews.isEmpty
            ? <String>[pack.thumbnailUrl]
            : <String>[
              for (final StickerPreview p in pack.previews.take(6)) p.thumbUrl,
            ];
    return LayoutBuilder(
      builder: (BuildContext context, BoxConstraints constraints) {
        final double width = constraints.maxWidth;
        if (pack.previews.isEmpty) {
          final double size = math.min(180, width * 0.6);
          return Center(
            child: _GridSticker(
              index: 0,
              url: urls.single,
              size: size,
              rotation: -0.06,
              reduce: reduce,
              coloured: _coloured(0, 1),
            ),
          );
        }
        const double gap = 8;
        final double cell = (width - 2 * gap) / 3;
        return Wrap(
          // One or two previews sit centred, not hugging the left edge.
          alignment: WrapAlignment.center,
          spacing: gap,
          runSpacing: gap,
          children: <Widget>[
            for (int i = 0; i < urls.length; i++)
              SizedBox.square(
                dimension: cell,
                child: Center(
                  child: _GridSticker(
                    index: i,
                    url: urls[i],
                    size: cell * 0.82,
                    rotation: _tilts[i],
                    reduce: reduce,
                    coloured: _coloured(i, urls.length),
                  ),
                ),
              ),
          ],
        );
      },
    );
  }

  bool _coloured(int index, int count) =>
      (pack.downloadStatus != PackDownloadStatus.downloading && pack.downloadStatus != PackDownloadStatus.failed) ||
      pack.downloadProgress >= (index + 1) / count;
}

/// Grid sticker that pops in (staggered) and wiggles briefly when tapped.
class _GridSticker extends StatefulWidget {
  const _GridSticker({
    required this.index,
    required this.url,
    required this.size,
    required this.rotation,
    required this.reduce,
    required this.coloured,
  });

  final int index;
  final String url;
  final double size;
  final double rotation;
  final bool reduce;
  final bool coloured;

  @override
  State<_GridSticker> createState() => _GridStickerState();
}

class _GridStickerState extends State<_GridSticker> {
  /// Bumped per tap; keys a fresh one-shot tween so each tap replays.
  int _wiggles = 0;

  @override
  Widget build(BuildContext context) {
    Widget sticker = DieCutSticker(
      url: widget.url,
      size: widget.size,
      rotation: widget.rotation,
    );
    if (!widget.coloured) {
      sticker = ColorFiltered(
        key: ValueKey<String>("restick.sheet.sticker.grey.${widget.index}"),
        colorFilter: const ColorFilter.mode(Colors.grey, BlendMode.saturation),
        child: sticker,
      );
    }
    if (_wiggles > 0 && !widget.reduce) {
      sticker = TweenAnimationBuilder<double>(
        key: ValueKey<int>(_wiggles),
        tween: Tween<double>(begin: 0, end: 1),
        duration: const Duration(milliseconds: 420),
        onEnd: () => setState(() => _wiggles = 0),
        builder:
            (BuildContext context, double v, Widget? child) => Transform.rotate(
              key: const ValueKey<String>("restick.sheet.wiggle"),
              angle: math.sin(v * math.pi * 4) * 0.12 * (1 - v),
              child: child,
            ),
        child: sticker,
      );
    }
    return GestureDetector(
      key: ValueKey<String>("restick.sheet.sticker.${widget.index}"),
      behavior: HitTestBehavior.opaque,
      onTap: widget.reduce ? null : () => setState(() => _wiggles++),
      // Stagger capped so all six land within 300ms.
      child: _PopIn(
        reduce: widget.reduce,
        delay: widget.index * 0.08,
        child: sticker,
      ),
    );
  }
}

/// One-shot scale-in over 300ms starting at [delay] (fraction of 300ms).
/// Renders [child] directly when animations are disabled.
class _PopIn extends StatelessWidget {
  const _PopIn({
    required this.reduce,
    required this.delay,
    required this.child,
  });

  final bool reduce;
  final double delay;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    if (reduce) {
      return child;
    }
    return TweenAnimationBuilder<double>(
      tween: Tween<double>(begin: 0, end: 1),
      duration: RestickMotion.routine,
      curve: Interval(
        delay,
        math.min(1, delay + 0.6),
        curve: Curves.easeOutBack,
      ),
      child: child,
      // Same structure after the tween ends so the child is never remounted.
      builder:
          (BuildContext context, double v, Widget? child) => Transform.scale(
            key: const ValueKey<String>("restick.sheet.pop"),
            scale: math.max(0, v),
            child: child,
          ),
    );
  }
}

/// Clips only the top edge; the other sides extend far enough to never cut.
class _TopEdgeClipper extends CustomClipper<Rect> {
  const _TopEdgeClipper();

  @override
  Rect getClip(Size size) => const Rect.fromLTRB(-100000, 0, 100000, 100000);

  @override
  bool shouldReclip(_TopEdgeClipper oldClipper) => false;
}
