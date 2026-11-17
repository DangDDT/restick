import "dart:math" as math;

import "package:material_ui/material_ui.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/domain/entities/sticker_preview.dart";
import "package:restick/src/presentation/theme/pack_tones.dart";
import "package:restick/src/presentation/theme/restick_brand.dart";
import "package:restick/src/presentation/theme/restick_strings.dart";
import "package:restick/src/presentation/theme/restick_theme.dart";
import "package:restick/src/presentation/widgets/pack_success_stamp.dart";
import "package:restick/src/presentation/widgets/die_cut_sticker.dart";
import "package:restick/src/presentation/widgets/download_button.dart";


// Rec. 709 luma weights; saturation 0 maps every pixel to its grey.
const ColorFilter _greyscale = ColorFilter.matrix(<double>[
  0.2126, 0.7152, 0.0722, 0, 0, //
  0.2126, 0.7152, 0.0722, 0, 0, //
  0.2126, 0.7152, 0.0722, 0, 0, //
  0, 0, 0, 1, 0,
]);

ColorFilter _saturation(double s) {
  if (s <= 0) {
    return _greyscale;
  }
  final double r = 0.2126 * (1 - s);
  final double g = 0.7152 * (1 - s);
  final double b = 0.0722 * (1 - s);
  return ColorFilter.matrix(<double>[
    r + s, g, b, 0, 0, //
    r, g + s, b, 0, 0, //
    r, g, b + s, 0, 0, //
    0, 0, 0, 1, 0,
  ]);
}

/// Up to three preview URLs for the strip; falls back to the pack thumbnail.
List<String> packStripUrls(StickerPack pack) {
  if (pack.previews.isEmpty) {
    return <String>[pack.thumbnailUrl];
  }
  return <String>[
    for (final StickerPreview p in pack.previews.take(3)) p.thumbUrl,
  ];
}

/// "Vừa lên kệ" row: a strip of three real stickers, name, meta and a compact
/// download button. While downloading the strip turns grey and each sticker
/// regains colour once `downloadProgress` reaches its share.
class PackCompactCard extends StatefulWidget {
  const PackCompactCard({
    required this.pack,
    required this.onOpen,
    required this.onDownload,
    this.strings = RestickStrings.vi,
    this.entranceIndex = 0,
    this.failureMessage = "",
    super.key,
  });

  final StickerPack pack;
  final VoidCallback onOpen;
  final VoidCallback onDownload;
  final RestickStrings strings;
  final String failureMessage;

  /// Position in the entrance stagger; negative skips the entrance (the
  /// card was already shown once). Read only when the card is first built.
  final int entranceIndex;

  @override
  State<PackCompactCard> createState() => _PackCompactCardState();
}

class _PackCompactCardState extends State<PackCompactCard> {
  // Latched: later rebuilds pass -1 for an id already seen.
  late final int _entrance = widget.entranceIndex;
  bool _pressed = false;

  // The compact owned chip is not tappable, so remove is never offered here.
  static void _noRemove() {}

  @override
  Widget build(BuildContext context) {
    final bool reduce = MediaQuery.disableAnimationsOf(context);
    final StickerPack pack = widget.pack;
    final PackTones tones = PackTones.of(pack);

    final Widget info = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: <Widget>[
        Text(
          pack.name,
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
          style: RestickBrand.text(
            size: 15,
            color: RestickTheme.of(context).ink,
            weight: FontWeight.w700,
            height: 1.15,
          ),
        ),
        Text(
          pack.downloadStatus == PackDownloadStatus.failed && widget.failureMessage.isNotEmpty
              ? widget.failureMessage
              : pack.isDownloaded ? widget.strings.readyInCollection
              : widget.strings.packMeta(pack.stickerCount, pack.author),
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: RestickBrand.text(
            size: 12,
            weight: FontWeight.w500,
            color: RestickTheme.of(context).mutedInk,
            height: 1.3,
          ),
        ),
      ],
    );
    final Widget button = PackDownloadButton(
      pack: pack,
      onDownload: widget.onDownload,
      onRemove: _noRemove,
      strings: widget.strings,
      compact: true,
    );
    final Widget strip = _PreviewStrip(
      pack: pack,
      tones: tones,
      reduce: reduce,
    );

    final Widget body = LayoutBuilder(
      builder: (BuildContext context, BoxConstraints constraints) {
        final bool stacked =
            constraints.maxWidth < 300 ||
            MediaQuery.textScalerOf(context).scale(14) > 20;
        final Widget row = Row(
          spacing: 10,
          children: <Widget>[
            strip,
            Expanded(child: info),
            if (!stacked) button,
          ],
        );
        if (!stacked) {
          return row;
        }
        return Column(
          crossAxisAlignment: CrossAxisAlignment.end,
          spacing: 8,
          children: <Widget>[row, button],
        );
      },
    );

    // No clipping anywhere up the tree: the button's burst paints outside.
    Widget card = DecoratedBox(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(24),
        boxShadow: <BoxShadow>[
          BoxShadow(
            color: RestickTheme.of(context).brand.withValues(alpha: 0x22 / 255),
            blurRadius: 12,
            offset: const Offset(0, 3),
          ),
        ],
      ),
      child: Material(
        color: Colors.white,
        borderRadius: BorderRadius.circular(24),
        child: InkWell(
          borderRadius: BorderRadius.circular(24),
          onTap: widget.onOpen,
          onHighlightChanged: (bool value) => setState(() => _pressed = value),
          child: Padding(padding: const EdgeInsets.all(10), child: body),
        ),
      ),
    );
    card = AnimatedScale(
      scale: _pressed && !reduce ? 0.97 : 1,
      duration: RestickMotion.fast,
      curve: RestickMotion.ease,
      child: card,
    );
    card = PackSuccessStamp(pack: pack, child: card);
    if (_entrance < 0 || reduce) {
      return card;
    }
    // Stagger capped so the whole entrance stays under 500ms.
    final int delay = math.min(_entrance, 4) * 40;
    final int total = RestickMotion.routine.inMilliseconds + delay;
    return TweenAnimationBuilder<double>(
      tween: Tween<double>(begin: 0, end: 1),
      duration: Duration(milliseconds: total),
      curve: Interval(delay / total, 1, curve: RestickMotion.ease),
      builder:
          (BuildContext context, double v, Widget? child) => Opacity(
            opacity: v,
            child: Transform.translate(
              offset: Offset(0, 18 * (1 - v)),
              child: child,
            ),
          ),
      child: card,
    );
  }
}

class _PreviewStrip extends StatelessWidget {
  const _PreviewStrip({
    required this.pack,
    required this.tones,
    required this.reduce,
  });

  final StickerPack pack;
  final PackTones tones;
  final bool reduce;

  static const double _size = 40;
  static const double _step = 30;
  static const List<double> _tilt = <double>[-0.10, 0.09, -0.07];

  @override
  Widget build(BuildContext context) {
    final List<String> urls = packStripUrls(pack);
    final int n = urls.length;
    final bool downloading =
        pack.downloadStatus == PackDownloadStatus.downloading;
    final bool failed = pack.downloadStatus == PackDownloadStatus.failed;
    final double progress = pack.downloadProgress.clamp(0, 1);

    return DecoratedBox(
      decoration: BoxDecoration(
        color: tones.background,
        borderRadius: BorderRadius.circular(18),
      ),
      child: Padding(
        padding: const EdgeInsets.all(6),
        child: SizedBox(
          width: _size + (n - 1) * _step,
          height: _size,
          child: Stack(
            clipBehavior: Clip.none,
            children: <Widget>[
              for (int i = 0; i < n; i++)
                Positioned(
                  left: i * _step,
                  top: 0,
                  child: _StripSlot(
                    index: i,
                    url: urls[i],
                    tilt: _tilt[i],
                    // Failed keeps the progress reached; anything else is
                    // full colour (not started, or owned).
                    coloured:
                        !(downloading || failed) || progress >= (i + 1) / n,
                    reduce: reduce,
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

class _StripSlot extends StatelessWidget {
  const _StripSlot({
    required this.index,
    required this.url,
    required this.tilt,
    required this.coloured,
    required this.reduce,
  });

  final int index;
  final String url;
  final double tilt;
  final bool coloured;
  final bool reduce;

  @override
  Widget build(BuildContext context) {
    final double target = coloured ? 1 : 0;
    final Widget sticker = DieCutSticker(
      url: url,
      size: _PreviewStrip._size,
      rotation: tilt,
    );
    return TweenAnimationBuilder<double>(
      tween: Tween<double>(end: target),
      duration:
          reduce
              ? Duration.zero
              : coloured
              ? RestickMotion.enter
              : RestickMotion.routine,
      curve: RestickMotion.ease,
      builder: (BuildContext context, double v, Widget? child) {
        // Fully coloured slots carry no filter layer (cheap in long lists).
        final Widget painted =
            v >= 1
                ? child!
                : ColorFiltered(
                  key: ValueKey<String>("restick.card.strip.grey.$index"),
                  colorFilter: _saturation(v),
                  child: child,
                );
        // Pop only while regaining colour, never while greying.
        final double pop = coloured && v < 1 ? 0.18 * math.sin(math.pi * v) : 0;
        return Transform.scale(scale: 1 + pop, child: painted);
      },
      child: sticker,
    );
  }
}
