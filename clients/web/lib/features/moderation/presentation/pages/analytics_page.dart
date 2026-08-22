import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick_web/features/moderation/presentation/cubit/analytics_cubit.dart";
import "package:restick_web/features/packs/domain/entities/pack.dart";
import "package:restick_web/l10n/app_localizations.dart";

class AnalyticsPage extends StatelessWidget {
  const AnalyticsPage({super.key});

  Color _barColor(String code) {
    switch (code) {
      case "copyright":
        return const Color(0xFFEF4444);
      case "quality":
        return const Color(0xFFF59E0B);
      case "duplicate":
        return const Color(0xFF6366F1);
      case "misleading":
        return const Color(0xFF8B5CF6);
      case "explicit":
        return const Color(0xFF64748B);
      default:
        return const Color(0xFF94A3B8);
    }
  }

  String _label(String code) {
    switch (code) {
      case "copyright":
        return "Copyright / IP";
      case "quality":
        return "Low Technical Quality";
      case "duplicate":
        return "Duplicate Artwork";
      case "misleading":
        return "Misleading Metadata";
      case "explicit":
        return "Explicit / Unsafe";
      default:
        return code;
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return BlocBuilder<AnalyticsCubit, AnalyticsState>(
      builder: (BuildContext context, AnalyticsState state) {
        final int total = state.items.fold<int>(
          0,
          (int sum, RejectionStat item) => sum + item.rejectionCount,
        );
        return Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            spacing: 16,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      spacing: 4,
                      children: [
                        Text(
                          l10n.titleAnalytics,
                          style: Theme.of(context).textTheme.headlineSmall,
                        ),
                        Text(
                          l10n.descAnalytics,
                          style: Theme.of(context).textTheme.bodyMedium,
                        ),
                      ],
                    ),
                  ),
                  IconButton.outlined(
                    tooltip: l10n.btnRefresh,
                    onPressed: () => context
                        .read<AnalyticsCubit>()
                        .load(forceRefresh: true),
                    icon: const Icon(Icons.refresh_rounded),
                  ),
                ],
              ),
              if (!state.hasLoaded && state.isRefreshing)
                const LinearProgressIndicator(minHeight: 2)
              else if (state.items.isEmpty)
                Text(l10n.emptyAnalytics)
              else
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(20),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      spacing: 16,
                      children: [
                        Text(
                          l10n.chartTitle,
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        ...state.items.map((RejectionStat item) {
                          final double ratio = total == 0
                              ? 0
                              : item.rejectionCount / total;
                          return Row(
                            spacing: 12,
                            children: [
                              SizedBox(
                                width: 180,
                                child: Text(
                                  _label(item.reasonCode),
                                  style: Theme.of(context).textTheme.bodySmall,
                                ),
                              ),
                              Expanded(
                                child: ClipRRect(
                                  borderRadius: BorderRadius.circular(8),
                                  child: LinearProgressIndicator(
                                    value: ratio,
                                    minHeight: 22,
                                    color: _barColor(item.reasonCode),
                                    backgroundColor: Theme.of(context)
                                        .colorScheme
                                        .surfaceContainerHighest,
                                  ),
                                ),
                              ),
                              Text(
                                "${(ratio * 100).round()}% (${item.rejectionCount})",
                                style: Theme.of(context).textTheme.labelMedium,
                              ),
                            ],
                          );
                        }),
                      ],
                    ),
                  ),
                ),
            ],
          ),
        );
      },
    );
  }
}
