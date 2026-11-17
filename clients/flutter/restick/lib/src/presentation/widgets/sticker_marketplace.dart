import "dart:async";

import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick/src/core/restick_client.dart";
import "package:restick/src/core/text/fold_vietnamese.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";
import "package:restick/src/domain/entities/pack_download_state.dart";
import "package:restick/src/presentation/cubit/marketplace_cubit.dart";
import "package:restick/src/presentation/scope/restick_scope.dart";
import "package:restick/src/presentation/theme/restick_brand.dart";
import "package:restick/src/presentation/theme/restick_strings.dart";
import "package:restick/src/presentation/theme/restick_theme.dart";
import "package:restick/src/presentation/widgets/pack_compact_card.dart";
import "package:restick/src/presentation/widgets/pack_detail_sheet.dart";
import "package:restick/src/presentation/widgets/pack_hero_carousel.dart";
import "package:restick/src/presentation/widgets/pack_skeleton.dart";
import "package:restick/src/presentation/widgets/restick_mascot.dart";


/// Idle loops pause while another route (or the detail sheet) covers us.
bool _routeIsCurrent(BuildContext context) =>
    ModalRoute.of(context)?.isCurrent ?? true;

class StickerMarketplace extends StatelessWidget {
  const StickerMarketplace({
    this.client,
    this.tag,
    this.strings = RestickStrings.vi,
    super.key,
  });

  final RestickClient? client;
  final String? tag;
  final RestickStrings strings;

  @override
  Widget build(BuildContext context) {
    final RestickClient resolved = client ?? RestickScope.of(context);
    return RepositoryProvider<RestickClient>.value(
      value: resolved,
      child: BlocProvider<MarketplaceCubit>(
        create: (_) => MarketplaceCubit(client: resolved)..load(tag: tag),
        child: _MarketplaceView(strings: strings),
      ),
    );
  }
}

class _MarketplaceView extends StatefulWidget {
  const _MarketplaceView({required this.strings});

  final RestickStrings strings;

  @override
  State<_MarketplaceView> createState() => _MarketplaceViewState();
}

class _MarketplaceViewState extends State<_MarketplaceView> {
  final TextEditingController _query = TextEditingController();

  /// Pack ids whose card already played its entrance (no replay on scroll).
  final Set<String> _entered = <String>{};
  bool _searching = false;
  String? _tag;
  final Set<String> _busy = <String>{};
  Timer? _cheer;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (MediaQuery.disableAnimationsOf(context)) {
      _cheer?.cancel();
      _cheer = null;
    }
  }

  /// False until the cubit first emits; shows the skeleton instead of an
  /// empty-state flash before the first load reports anything.
  bool _settled = false;

  final GlobalKey _heroKey = GlobalKey();

  /// Whether any part of the hero carousel is inside the list viewport.
  bool _heroVisible = true;

  // A SliverToBoxAdapter child stays mounted (and TickerMode on) after it
  // scrolls away, so the carousel is told explicitly when it is off-screen.
  bool _trackHero(
    int depth,
    ScrollMetrics metrics,
    BuildContext? scrollable,
  ) {
    if (depth != 0 || metrics.axis != Axis.vertical) {
      return false;
    }
    final RenderObject? hero = _heroKey.currentContext?.findRenderObject();
    final RenderObject? viewport = scrollable?.findRenderObject();
    if (hero is! RenderBox || viewport == null || !hero.attached) {
      return false;
    }
    final double bottom =
        hero.localToGlobal(Offset.zero, ancestor: viewport).dy +
        hero.size.height;
    final bool visible = bottom > 0;
    if (visible != _heroVisible) {
      setState(() => _heroVisible = visible);
    }
    return false;
  }

  RestickStrings get _s => widget.strings;

  @override
  void dispose() {
    _query.dispose();
    _cheer?.cancel();
    super.dispose();
  }

  /// Set while a detail sheet is open so rapid taps do not stack sheets.
  bool _detailOpen = false;

  Future<void> _openPackDetail(StickerPack pack) async {
    if (_detailOpen) {
      return;
    }
    _detailOpen = true;
    try {
      await showPackDetailSheet(
        context,
        cubit: context.read<MarketplaceCubit>(),
        packId: pack.id,
        strings: _s,
      );
    } finally {
      _detailOpen = false;
    }
  }

  void _toggleSearch() {
    setState(() {
      _searching = !_searching;
      if (!_searching) {
        _query.clear();
      }
    });
  }

  bool _matches(StickerPack pack, String query, String? tag) {
    if (tag != null && !pack.tags.contains(tag)) {
      return false;
    }
    return query.isEmpty ||
        foldVietnamese(pack.name).contains(query) ||
        foldVietnamese(pack.author).contains(query) ||
        pack.tags.any((String t) => foldVietnamese(t).contains(query));
  }

  @override
  Widget build(BuildContext context) {
    return ColoredBox(
      color: RestickTheme.of(context).canvas,
      child: BlocConsumer<MarketplaceCubit, MarketplaceState>(
        // Any emission means the first load has started; a builder alone
        // can miss it when loading and loaded land in the same frame.
        listener: (BuildContext context, MarketplaceState state) {
          final bool completed = state.packs.any((StickerPack p) =>
              _busy.contains(p.id) && p.isDownloaded);
          _busy
            ..clear()
            ..addAll(state.packs.where((StickerPack p) =>
                p.downloadStatus == PackDownloadStatus.downloading).map((StickerPack p) => p.id));
          if (completed && !MediaQuery.disableAnimationsOf(context)) {
            _cheer?.cancel();
            setState(() {
              _cheer = Timer(const Duration(milliseconds: 900), () {
                if (mounted) setState(() => _cheer = null);
              });
            });
          }
          if (!_settled) {
            setState(() => _settled = true);
          }
        },
        builder: (BuildContext context, MarketplaceState state) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: <Widget>[
              _header(context),
              Expanded(child: _body(context, state)),
            ],
          );
        },
      ),
    );
  }

  Widget _header(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 8, 4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        spacing: 6,
        children: <Widget>[
          Row(
            spacing: 8,
            children: <Widget>[
              RestickMascot(size: 38, blink: _routeIsCurrent(context),
                mood: _cheer == null ? RestickMood.happy : RestickMood.cheer),
              Expanded(
                child: Text.rich(
                  TextSpan(
                    text: "Re",
                    children: <InlineSpan>[
                      TextSpan(
                        text: "stick",
                        style: RestickBrand.text(
                          size: 21,
                          weight: FontWeight.w700,
                          color: RestickBrand.deepPink,
                        ),
                      ),
                    ],
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: RestickBrand.text(size: 21, weight: FontWeight.w700, color: RestickTheme.of(context).ink),
                ),
              ),
              IconButton(
                key: const ValueKey<String>("marketplace.search.open"),
                tooltip: _s.searchHint,
                onPressed: _toggleSearch,
                color: RestickTheme.of(context).ink,
                icon: Icon(
                  _searching ? Icons.close_rounded : Icons.search_rounded,
                ),
              ),
            ],
          ),
          if (_searching)
            Padding(
              padding: const EdgeInsets.only(right: 8),
              child: TextField(
                key: const ValueKey<String>("marketplace.search"),
                controller: _query,
                autofocus: true,
                textInputAction: TextInputAction.search,
                onChanged: (_) => setState(() {}),
                style: RestickBrand.text(weight: FontWeight.w500, color: RestickTheme.of(context).ink),
                decoration: InputDecoration(
                  isDense: true,
                  filled: true,
                  fillColor: Colors.white,
                  hintText: _s.searchHint,
                  hintStyle: RestickBrand.text(
                    weight: FontWeight.w500,
                    color: RestickTheme.of(context).mutedInk,
                  ),
                  prefixIcon: const Icon(Icons.search_rounded, size: 20),
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(99),
                    borderSide: BorderSide.none,
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }

  Widget _body(BuildContext context, MarketplaceState state) {
    final MarketplaceCubit cubit = context.read<MarketplaceCubit>();
    if (state.packs.isEmpty && (state.isLoading || !_settled)) {
      return const _MarketplaceSkeleton();
    }
    if (state.packs.isEmpty) {
      final bool failed = state.failure != null;
      return RefreshIndicator(
        color: RestickBrand.deepPink,
        onRefresh: cubit.load,
        child: CustomScrollView(
          physics: const AlwaysScrollableScrollPhysics(),
          slivers: <Widget>[
            SliverFillRemaining(
              hasScrollBody: false,
              child: _MarketplaceMessage(
                mood: failed ? RestickMood.sad : RestickMood.happy,
                title: failed ? _s.errorTitle : _s.emptyTitle,
                detail: state.failure?.message,
                retryLabel: failed ? _s.retry : null,
                onRetry: failed ? () => unawaited(cubit.load()) : null,
              ),
            ),
          ],
        ),
      );
    }

    final List<String> tags = <String>{
      for (final StickerPack pack in state.packs) ...pack.tags,
    }.toList();
    final String? tag = tags.contains(_tag) ? _tag : null;
    final String query = _searching ? foldVietnamese(_query.text.trim()) : "";
    final bool filtering = tag != null || query.isNotEmpty;
    // Sublists of the cubit's own instances: the download button relies on
    // pack identity to detect updates.
    final List<StickerPack> visible =
        filtering
            ? state.packs
                .where((StickerPack p) => _matches(p, query, tag))
                .toList()
            : state.packs;
    final List<StickerPack> hero =
        filtering ? const <StickerPack>[] : visible.take(3).toList();
    final List<StickerPack> rest =
        filtering ? visible : visible.skip(3).toList();

    final Widget list = CustomScrollView(
      physics: const AlwaysScrollableScrollPhysics(),
      slivers: <Widget>[
        if (tags.isNotEmpty)
          SliverToBoxAdapter(
            child: SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 10),
              child: Row(
                spacing: 6,
                children: <Widget>[
                  for (final String t in tags)
                    _TagChip(
                      key: ValueKey<String>("marketplace.chip.$t"),
                      label: t,
                      selected: t == tag,
                      onTap: () => setState(() => _tag = t == tag ? null : t),
                    ),
                ],
              ),
            ),
          ),
        if (hero.isNotEmpty)
          SliverToBoxAdapter(
            child: PackHeroCarousel(
              key: _heroKey,
              active: _heroVisible,
              packs: hero,
              strings: _s,
              downloads: state.downloads,
              onOpen: (StickerPack pack) => unawaited(_openPackDetail(pack)),
              onDownload: (String id) => unawaited(cubit.download(id)),
            ),
          ),
        if (!filtering && rest.isNotEmpty)
          SliverToBoxAdapter(
            child: Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 8),
              child: Text(
                _s.newShelf,
                style: RestickBrand.text(size: 16, weight: FontWeight.w700, color: RestickTheme.of(context).ink),
              ),
            ),
          ),
        if (filtering && rest.isEmpty)
          SliverFillRemaining(
            hasScrollBody: false,
            child: _MarketplaceMessage(
              mood: RestickMood.happy,
              title: _s.emptyTitle,
            ),
          ),
        SliverPadding(
          // Bottom room so the last card's download burst is not cut.
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 40),
          sliver: SliverList.builder(
            itemCount: rest.length,
            itemBuilder: (BuildContext context, int index) {
              final StickerPack pack = rest[index];
              return Padding(
                padding: const EdgeInsets.only(bottom: 10),
                child: PackCompactCard(
                  key: ValueKey<String>(pack.id),
                  pack: pack,
                  strings: _s,
                  failureMessage: state.downloads[pack.id]?.message ?? "",
                  entranceIndex: _entered.add(pack.id) ? index : -1,
                  onOpen: () => unawaited(_openPackDetail(pack)),
                  onDownload: () => unawaited(cubit.download(pack.id)),
                ),
              );
            },
          ),
        ),
      ],
    );
    return RefreshIndicator(
      color: RestickBrand.deepPink,
      onRefresh: cubit.load,
      // Metrics changes (filter cleared, reload) re-check visibility too.
      child: NotificationListener<ScrollMetricsNotification>(
        onNotification:
            (ScrollMetricsNotification n) =>
                _trackHero(n.depth, n.metrics, n.context),
        child: NotificationListener<ScrollNotification>(
          onNotification:
              (ScrollNotification n) =>
                  _trackHero(n.depth, n.metrics, n.context),
          child: list,
        ),
      ),
    );
  }
}

class _TagChip extends StatelessWidget {
  const _TagChip({
    required this.label,
    required this.selected,
    required this.onTap,
    super.key,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      selected: selected,
      child: GestureDetector(
        onTap: onTap,
        behavior: HitTestBehavior.opaque,
        child: ConstrainedBox(
          constraints: const BoxConstraints(minWidth: 48, minHeight: 48),
          child: Center(
            child: AnimatedContainer(
          duration:
              MediaQuery.disableAnimationsOf(context)
                  ? Duration.zero
                  : RestickMotion.fast,
          padding: const EdgeInsets.symmetric(horizontal: 13, vertical: 6),
          decoration: BoxDecoration(
            color: selected ? RestickTheme.of(context).ink : Colors.white,
            borderRadius: BorderRadius.circular(99),
          ),
          child: Text(
            label,
            style: RestickBrand.text(
              size: 12,
              color: selected ? Colors.white : RestickTheme.of(context).mutedInk,
            ),
          ),
        ),
          ),
        ),
      ),
    );
  }
}

class _MarketplaceMessage extends StatelessWidget {
  const _MarketplaceMessage({
    required this.mood,
    required this.title,
    this.detail,
    this.retryLabel,
    this.onRetry,
  });

  final RestickMood mood;
  final String title;
  final String? detail;
  final String? retryLabel;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final String? detailText = detail;
    final VoidCallback? retry = onRetry;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          spacing: 10,
          children: <Widget>[
            RestickMascot(
              size: 96,
              mood: mood,
              blink: _routeIsCurrent(context),
            ),
            Text(
              title,
              textAlign: TextAlign.center,
              style: RestickBrand.text(size: 17, weight: FontWeight.w700, color: RestickTheme.of(context).ink),
            ),
            if (detailText != null)
              Text(
                detailText,
                textAlign: TextAlign.center,
                style: RestickBrand.text(
                  size: 13,
                  weight: FontWeight.w500,
                  color: RestickTheme.of(context).mutedInk,
                ),
              ),
            if (retry != null)
              FilledButton(
                key: const ValueKey<String>("marketplace.retry"),
                onPressed: retry,
                style: FilledButton.styleFrom(
                  backgroundColor: RestickBrand.deepPink,
                  foregroundColor: Colors.white,
                ),
                child: Text(
                  retryLabel ?? "",
                  style: RestickBrand.text(
                    weight: FontWeight.w700,
                    color: Colors.white,
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

/// Pastel shimmer in the shape of the page (hero card, dots, three rows).
class _MarketplaceSkeleton extends StatelessWidget {
  const _MarketplaceSkeleton();

  @override
  Widget build(BuildContext context) {
    final double width = MediaQuery.sizeOf(context).width;
    return ListView(
      physics: const NeverScrollableScrollPhysics(),
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
      children: <Widget>[
        Align(
          alignment: Alignment.centerLeft,
          child: PackSkeleton(
            width: width * 0.72,
            height: 262,
            radius: 28,
            color: RestickBrand.packPastels[1],
          ),
        ),
        const SizedBox(height: 24),
        const PackSkeleton(width: 120, height: 18, radius: 9),
        const SizedBox(height: 12),
        for (int i = 0; i < 3; i++)
          Padding(
            padding: const EdgeInsets.only(bottom: 10),
            child: PackSkeleton(
              height: 72,
              radius: 24,
              color: RestickBrand.packPastels[(i + 2) % 5],
            ),
          ),
      ],
    );
  }
}
