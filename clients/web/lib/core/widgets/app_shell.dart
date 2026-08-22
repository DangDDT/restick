import "package:auto_size_text/auto_size_text.dart";
import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:go_router/go_router.dart";
import "package:responsive_framework/responsive_framework.dart";
import "package:restick_web/core/theme/theme_cubit.dart";
import "package:restick_web/core/widgets/authenticated_image.dart";
import "package:restick_web/core/widgets/status_pill.dart";
import "package:restick_web/features/auth/presentation/cubit/auth_cubit.dart";
import "package:restick_web/features/packs/domain/entities/pack.dart";
import "package:restick_web/features/preview/presentation/widgets/device_preview_panel.dart";
import "package:restick_web/l10n/app_localizations.dart";

class AppShell extends StatelessWidget {
  const AppShell({required this.child, super.key});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final bool isNarrow = ResponsiveBreakpoints.of(context).smallerThan(TABLET);
    final String path = GoRouterState.of(context).uri.path;
    final bool showPreview = ResponsiveBreakpoints.of(
          context,
        ).largerThan(TABLET) &&
        !path.startsWith("/api-docs");
    if (isNarrow) {
      return _MobileShell(child: child);
    }
    return _DesktopShell(showPreview: showPreview, child: child);
  }
}

class _DesktopShell extends StatelessWidget {
  const _DesktopShell({required this.showPreview, required this.child});

  final bool showPreview;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final ColorScheme colors = Theme.of(context).colorScheme;
    return Scaffold(
      backgroundColor: colors.surfaceContainerLowest,
      body: Row(
        children: [
          const AppSidebar(),
          Expanded(
            child: Row(
              children: [
                Expanded(child: ClipRect(child: child)),
                if (showPreview) const DevicePreviewPanel(),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _MobileShell extends StatelessWidget {
  const _MobileShell({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      drawer: const Drawer(child: AppSidebar()),
      appBar: AppBar(
        title: Text(AppLocalizations.of(context).appTitle),
      ),
      body: ClipRect(child: child),
    );
  }
}

class AppSidebar extends StatelessWidget {
  const AppSidebar({super.key});

  @override
  Widget build(BuildContext context) {
    final ColorScheme colors = Theme.of(context).colorScheme;
    final AppLocalizations l10n = AppLocalizations.of(context);
    final AuthState auth = context.watch<AuthCubit>().state;
    final String path = GoRouterState.of(context).uri.path;
    return Container(
      width: 260,
      decoration: BoxDecoration(
        color: colors.surface.withValues(alpha: 0.85),
        border: Border(right: BorderSide(color: colors.outline)),
      ),
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            spacing: 12,
            children: [
              Container(
                width: 36,
                height: 36,
                decoration: BoxDecoration(
                  color: colors.primary,
                  borderRadius: BorderRadius.circular(10),
                ),
                child: Icon(Icons.layers, color: colors.onPrimary),
              ),
              AutoSizeText(
                l10n.appTitle.toUpperCase(),
                style: Theme.of(context).textTheme.titleLarge,
                maxLines: 1,
              ),
            ],
          ),
          const SizedBox(height: 28),
          Expanded(
            child: ListView(
              children: [
          Text(
            l10n.navContribHeader,
            style: Theme.of(context).textTheme.labelSmall?.copyWith(
              color: colors.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 8),
          _NavItem(
            label: l10n.navMyPacks,
            icon: Icons.grid_view_rounded,
            selected:
                path == "/packs" ||
                (path.startsWith("/packs/") && !path.startsWith("/packs/new")),
            onTap: () => context.go("/packs"),
          ),
          _NavItem(
            label: l10n.navUploadPack,
            icon: Icons.add_box_outlined,
            selected: path.startsWith("/packs/new"),
            onTap: () => context.go("/packs/new"),
          ),
          _NavItem(
            label: l10n.navApiKeys,
            icon: Icons.vpn_key_outlined,
            selected: path.startsWith("/keys"),
            onTap: () => context.go("/keys"),
          ),
          _NavItem(
            label: l10n.navApiDocs,
            icon: Icons.menu_book_outlined,
            selected: path.startsWith("/api-docs"),
            onTap: () => context.go("/api-docs"),
          ),
          if (auth.isAdmin) ...[
            const SizedBox(height: 20),
            Text(
              l10n.navAdminHeader,
              style: Theme.of(context).textTheme.labelSmall?.copyWith(
                color: colors.tertiary,
              ),
            ),
            const SizedBox(height: 8),
            _NavItem(
              label: l10n.navQueue,
              icon: Icons.rate_review_outlined,
              selected: path.startsWith("/admin/queue"),
              onTap: () => context.go("/admin/queue"),
            ),
            _NavItem(
              label: l10n.navCatalog,
              icon: Icons.public_outlined,
              selected: path.startsWith("/admin/published"),
              onTap: () => context.go("/admin/published"),
            ),
          ],
              ],
            ),
          ),
          const _UserCard(),
        ],
      ),
    );
  }
}

class _NavItem extends StatelessWidget {
  const _NavItem({
    required this.label,
    required this.icon,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final IconData icon;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final ColorScheme colors = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Material(
        color: selected
            ? colors.primary.withValues(alpha: 0.12)
            : Colors.transparent,
        borderRadius: BorderRadius.circular(10),
        child: InkWell(
          onTap: onTap,
          borderRadius: BorderRadius.circular(10),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            child: Row(
              spacing: 10,
              children: [
                Icon(
                  icon,
                  color: selected ? colors.primary : colors.onSurfaceVariant,
                ),
                Expanded(
                  child: Text(
                    label,
                    style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                      color: selected ? colors.primary : colors.onSurface,
                      fontWeight: selected ? FontWeight.w600 : FontWeight.w500,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _UserCard extends StatelessWidget {
  const _UserCard();

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final AuthState auth = context.watch<AuthCubit>().state;
    final ThemeMode themeMode = context.watch<ThemeCubit>().state;
    final LocaleState locale = context.watch<LocaleCubit>().state;
    final ColorScheme colors = Theme.of(context).colorScheme;
    final String name = auth.user?.displayName ?? "User";
    final String initials = name.isNotEmpty
        ? name
              .trim()
              .split(RegExp(r"\s+"))
              .take(2)
              .map((String p) => p[0].toUpperCase())
              .join()
        : "U";
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: colors.surfaceContainerHighest.withValues(alpha: 0.35),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: colors.outline),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        spacing: 10,
        children: [
          Row(
            spacing: 10,
            children: [
              CircleAvatar(
                backgroundColor: colors.primary,
                foregroundColor: colors.onPrimary,
                child: Text(initials),
              ),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(name, style: Theme.of(context).textTheme.titleSmall),
                    Text(
                      auth.isAdmin ? l10n.roleAdmin : l10n.roleContributor,
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  ],
                ),
              ),
            ],
          ),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              IconButton.outlined(
                tooltip: themeMode == ThemeMode.dark
                    ? l10n.themeLight
                    : l10n.themeDark,
                onPressed: () => context.read<ThemeCubit>().toggle(),
                icon: Icon(
                  themeMode == ThemeMode.dark
                      ? Icons.light_mode_outlined
                      : Icons.dark_mode_outlined,
                ),
              ),
              IconButton.outlined(
                tooltip: l10n.language,
                onPressed: () => context.read<LocaleCubit>().toggle(),
                icon: Text(
                  locale.locale.languageCode == "vi" ? "EN" : "VI",
                  style: Theme.of(context).textTheme.labelLarge,
                ),
              ),
              IconButton.outlined(
                tooltip: l10n.signOut,
                onPressed: () async {
                  await context.read<AuthCubit>().logout();
                  if (context.mounted) {
                    context.go("/login");
                  }
                },
                icon: const Icon(Icons.logout_rounded),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class PackCard extends StatelessWidget {
  const PackCard({
    required this.pack,
    required this.onTap,
    this.onDelete,
    this.trailing,
    super.key,
  });

  final PackSummary pack;
  final VoidCallback onTap;
  final VoidCallback? onDelete;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    return Card(
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            spacing: 12,
            children: [
              AspectRatio(
                aspectRatio: 1.4,
                child: DecoratedBox(
                  decoration: BoxDecoration(
                    color: Theme.of(context).colorScheme.surfaceContainerHighest
                        .withValues(alpha: 0.4),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: pack.thumbnailUrl.isEmpty
                      ? const Icon(Icons.image_outlined)
                      : AuthenticatedImage(url: pack.thumbnailUrl),
                ),
              ),
              Row(
                children: [
                  Expanded(child: StatusPill(status: pack.status)),
                  if (onDelete != null)
                    PopupMenuButton<String>(
                      tooltip: l10n.menuDelete,
                      onSelected: (String value) {
                        if (value == "delete") {
                          onDelete!();
                        }
                      },
                      itemBuilder: (BuildContext context) {
                        return <PopupMenuEntry<String>>[
                          PopupMenuItem<String>(
                            value: "delete",
                            child: Row(
                              spacing: 8,
                              children: [
                                const Icon(Icons.delete_outline, size: 18),
                                Text(l10n.menuDelete),
                              ],
                            ),
                          ),
                        ];
                      },
                      icon: const Icon(Icons.more_vert, size: 20),
                    ),
                ],
              ),
              Text(pack.name, style: Theme.of(context).textTheme.titleMedium),
              Text(
                l10n.packSlugMeta(pack.slug, pack.stickerCount),
                style: Theme.of(context).textTheme.bodySmall,
              ),
              if (trailing != null) trailing!,
            ],
          ),
        ),
      ),
    );
  }
}
