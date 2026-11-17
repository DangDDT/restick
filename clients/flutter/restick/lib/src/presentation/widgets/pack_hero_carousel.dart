import "dart:math" as math;

import "package:material_ui/material_ui.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/presentation/theme/pack_tones.dart";
import "package:restick/src/presentation/theme/restick_brand.dart";
import "package:restick/src/presentation/theme/restick_strings.dart";
import "package:restick/src/presentation/theme/restick_theme.dart";
import "package:restick/src/presentation/widgets/pack_success_stamp.dart";
import "package:restick/src/presentation/widgets/die_cut_sticker.dart";
import "package:restick/src/presentation/widgets/download_button.dart";

const Color _dotIdle = Color(0xFFE3CFE0);

/// Large-sticker URL of a hero card: first preview's 512px file, else the
/// pack thumbnail (older servers send no previews).
String packHeroUrl(StickerPack pack) =>
    pack.previews.isEmpty ? pack.thumbnailUrl : pack.previews.first.fileUrl;

/// Swipeable big cards for the first packs: the neighbour card peeks in, the
/// big sticker shifts with the page (parallax) and floats on the current
/// page only; dots stretch on the current page.
class PackHeroCarousel extends StatefulWidget {
  const PackHeroCarousel({
    required this.packs,
    required this.onOpen,
    required this.onDownload,
    this.strings = RestickStrings.vi,
    this.active = true,
    this.downloads = const <String, PackDownloadState>{},
    super.key,
  });

  final List<StickerPack> packs;
  final void Function(StickerPack) onOpen;
  final void Function(String) onDownload;
  final RestickStrings strings;
  final Map<String, PackDownloadState> downloads;

  /// False while the carousel is scrolled out of view: the float stops.
  final bool active;

  @override
  State<PackHeroCarousel> createState() => _PackHeroCarouselState();
}

class _PackHeroCarouselState extends State<PackHeroCarousel>
    with TickerProviderStateMixin {
  final PageController _controller = PageController(viewportFraction: 0.78);
  int _page = 0;

  // Created only while idle motion is allowed (recreated after a covering
  // route pops, hence TickerProviderStateMixin); TickerMode mutes it too.
  AnimationController? _float;

  List<StickerPack> get _packs => widget.packs.take(3).toList();

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _syncFloat();
  }

  @override
  void didUpdateWidget(PackHeroCarousel oldWidget) {
    super.didUpdateWidget(oldWidget);
    final int last = math.max(0, _packs.length - 1);
    if (_page > last) {
      _page = last;
    }
    _syncFloat();
  }

  void _syncFloat() {
    final bool want =
        widget.active &&
        _packs.isNotEmpty &&
        !MediaQuery.disableAnimationsOf(context) &&
        (ModalRoute.of(context)?.isCurrent ?? true);
    if (want && _float == null) {
      _float = AnimationController(
        vsync: this,
        duration: const Duration(milliseconds: 4500),
      )..repeat();
    } else if (!want && _float != null) {
      _float!.dispose();
      _float = null;
    }
  }

  @override
  void dispose() {
    _float?.dispose();
    _controller.dispose();
    super.dispose();
  }

  double _pagePosition() {
    return _controller.hasClients && _controller.position.haveDimensions
        ? _controller.page ?? _page.toDouble()
        : _page.toDouble();
  }

  @override
  Widget build(BuildContext context) {
    final List<StickerPack> packs = _packs;
    if (packs.isEmpty) {
      return const SizedBox.shrink();
    }
    final TextScaler scaler = MediaQuery.textScalerOf(context);
    final bool reduce = MediaQuery.disableAnimationsOf(context);
    // Text band grows with the text scale so two-line names never overflow.
    final double textBand =
        2 * scaler.scale(20) * 1.1 + 6 + math.max(36, scaler.scale(12) * 1.3);
    final double height = math.max(262, 32 + 120 + textBand);

    return Column(
      mainAxisSize: MainAxisSize.min,
      children: <Widget>[
        SizedBox(
          height: height + 16,
          child: Padding(
            padding: const EdgeInsets.only(left: 10),
            child: PageView.builder(
              controller: _controller,
              // The download burst paints outside the card; do not cut it.
              clipBehavior: Clip.none,
              padEnds: false,
              itemCount: packs.length,
              onPageChanged: (int page) => setState(() => _page = page),
              itemBuilder: (BuildContext context, int index) {
                return AnimatedBuilder(
                  animation: _controller,
                  builder: (BuildContext context, Widget? child) {
                    final double delta = reduce ? 0 : _pagePosition() - index;
                    return Transform.scale(scale: 1 - 0.06 * delta.abs().clamp(0, 1), child: child);
                  },
                  child: PackSuccessStamp(
                    pack: packs[index],
                    child: _HeroCard(
                      pack: packs[index],
                      strings: widget.strings,
                      failureMessage: widget.downloads[packs[index].id]?.message ?? "",
                      controller: _controller,
                      index: index,
                      float: index == _page ? _float : null,
                      onTap: () {
                        if (index == _page) {
                          widget.onOpen(packs[index]);
                        } else {
                          _controller.animateToPage(index, duration: RestickMotion.routine, curve: RestickMotion.ease);
                        }
                      },
                      onDownload: () => widget.onDownload(packs[index].id),
                    ),
                  ),
                );
              },
            ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.symmetric(vertical: 8),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            spacing: 5,
            children: <Widget>[
              for (int i = 0; i < packs.length; i++)
                AnimatedContainer(
                  key: ValueKey<String>("restick.hero.dot.$i"),
                  duration: reduce ? Duration.zero : RestickMotion.routine,
                  curve: RestickMotion.ease,
                  width: i == _page ? 18 : 6,
                  height: 6,
                  decoration: BoxDecoration(
                    color: i == _page ? RestickBrand.deepPink : _dotIdle,
                    borderRadius: BorderRadius.circular(9),
                  ),
                ),
            ],
          ),
        ),
      ],
    );
  }
}

class _HeroCard extends StatelessWidget {
  const _HeroCard({
    required this.pack,
    required this.strings,
    required this.controller,
    required this.index,
    required this.failureMessage,
    required this.float,
    required this.onTap,
    required this.onDownload,
  });

  final StickerPack pack;
  final RestickStrings strings;

  final PageController controller;
  final int index;
  final String failureMessage;
  final Animation<double>? float;
  final VoidCallback onTap;
  final VoidCallback onDownload;

  // The compact owned chip is not tappable, so remove is never offered here.
  static void _noRemove() {}

  @override
  Widget build(BuildContext context) {
    final PackTones tones = PackTones.of(pack);
    final bool reduce = MediaQuery.disableAnimationsOf(context);

    Widget sticker = LayoutBuilder(
      builder: (BuildContext context, BoxConstraints constraints) {
        final double size = math.min(
          170,
          math.min(constraints.maxWidth, constraints.maxHeight) * 0.92,
        );
        return Center(
          child: DieCutSticker(
            url: packHeroUrl(pack),
            size: size,
            rotation: -0.1,
          ),
        );
      },
    );
    final Animation<double>? floating = float;
    if (floating != null) {
      sticker = AnimatedBuilder(
        animation: floating,
        builder: (BuildContext context, Widget? child) {
          final double phase = 2 * math.pi * floating.value;
          return Transform(
            key: const ValueKey<String>("restick.hero.float"),
            alignment: Alignment.center,
            transform: Matrix4.translationValues(0, math.sin(phase) * 6, 0)
              ..rotateZ(math.sin(phase + 1) * 0.03),
            child: child,
          );
        },
        child: sticker,
      );
    }

    final Widget content = Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 14),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        spacing: 6,
        children: <Widget>[
          Expanded(
            child: RepaintBoundary(
              child: AnimatedBuilder(
                animation: controller,
                builder: (BuildContext context, Widget? child) {
                  final double page = controller.hasClients && controller.position.haveDimensions
                      ? controller.page ?? index.toDouble() : index.toDouble();
                  return Transform.translate(offset: Offset(reduce ? 0 : -(page - index) * 40, 0), child: child);
                },
                child: sticker,
              ),
            ),
          ),
          Text(
            pack.name,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: RestickBrand.text(
              size: 20,
              color: RestickTheme.of(context).ink,
              weight: FontWeight.w700,
              height: 1.1,
            ),
          ),
          Row(
            spacing: 8,
            children: <Widget>[
              Expanded(
                child: Text(
                  pack.downloadStatus == PackDownloadStatus.failed && failureMessage.isNotEmpty
                      ? failureMessage
                      : pack.isDownloaded ? strings.readyInCollection
                      : strings.packMeta(pack.stickerCount, pack.author),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: RestickBrand.text(
                    size: 12,
                    weight: FontWeight.w500,
                    color: RestickTheme.of(context).mutedInk,
                    height: 1.3,
                  ),
                ),
              ),
              PackDownloadButton(
                pack: pack,
                onDownload: onDownload,
                onRemove: _noRemove,
                strings: strings,
                compact: true,
              ),
            ],
          ),
        ],
      ),
    );

    return Padding(
      padding: const EdgeInsets.fromLTRB(6, 8, 6, 8),
      child: GestureDetector(
          behavior: HitTestBehavior.opaque,
          onTap: onTap,
          // Only the background is rounded; nothing clips the button row.
          child: Stack(
            clipBehavior: Clip.none,
            children: <Widget>[
              Positioned.fill(
                child: DecoratedBox(
                  decoration: BoxDecoration(
                    color: tones.background,
                    borderRadius: BorderRadius.circular(28),
                  ),
                ),
              ),
              content,
            ],
          ),
        ),
    );
  }
}
