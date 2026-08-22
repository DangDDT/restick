import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:get_it/get_it.dart";
import "package:go_router/go_router.dart";
import "package:restick_web/core/router/page_transitions.dart";
import "package:restick_web/core/widgets/app_shell.dart";
import "package:restick_web/features/api_keys/presentation/cubit/api_key_cubit.dart";
import "package:restick_web/features/api_keys/presentation/pages/api_keys_page.dart";
import "package:restick_web/features/auth/presentation/cubit/auth_cubit.dart";
import "package:restick_web/features/auth/presentation/pages/login_page.dart";
import "package:restick_web/features/auth/presentation/pages/register_page.dart";
import "package:restick_web/features/moderation/presentation/cubit/analytics_cubit.dart";
import "package:restick_web/features/moderation/presentation/cubit/published_catalog_cubit.dart";
import "package:restick_web/features/moderation/presentation/cubit/review_detail_cubit.dart";
import "package:restick_web/features/moderation/presentation/cubit/review_queue_cubit.dart";
import "package:restick_web/features/moderation/presentation/pages/analytics_page.dart";
import "package:restick_web/features/moderation/presentation/pages/api_docs_page.dart";
import "package:restick_web/features/moderation/presentation/pages/published_catalog_page.dart";
import "package:restick_web/features/moderation/presentation/pages/review_detail_page.dart";
import "package:restick_web/features/moderation/presentation/pages/review_queue_page.dart";
import "package:restick_web/features/packs/presentation/cubit/pack_detail_cubit.dart";
import "package:restick_web/features/packs/presentation/cubit/pack_list_cubit.dart";
import "package:restick_web/features/packs/presentation/cubit/pack_upload_cubit.dart";
import "package:restick_web/features/packs/presentation/pages/my_packs_page.dart";
import "package:restick_web/features/packs/presentation/pages/new_pack_page.dart";
import "package:restick_web/features/packs/presentation/pages/pack_detail_page.dart";

GoRouter createAppRouter(AuthCubit authCubit) {
  return GoRouter(
    initialLocation: "/packs",
    refreshListenable: authCubit.listenable,
    redirect: (BuildContext context, GoRouterState state) {
      final AuthState auth = authCubit.state;
      if (!auth.isReady) {
        return null;
      }
      final bool onAuth =
          state.uri.path == "/login" || state.uri.path == "/register";
      if (!auth.isSignedIn && !onAuth) {
        return "/login";
      }
      if (auth.isSignedIn && onAuth) {
        return "/packs";
      }
      if (state.uri.path.startsWith("/admin") && !auth.isAdmin) {
        return "/packs";
      }
      return null;
    },
    routes: <RouteBase>[
      GoRoute(
        path: "/login",
        pageBuilder: (BuildContext context, GoRouterState state) {
          return buildAppPage(
            key: state.pageKey,
            child: const LoginPage(),
          );
        },
      ),
      GoRoute(
        path: "/register",
        pageBuilder: (BuildContext context, GoRouterState state) {
          return buildAppPage(
            key: state.pageKey,
            child: const RegisterPage(),
          );
        },
      ),
      ShellRoute(
        builder: (BuildContext context, GoRouterState state, Widget child) {
          return AppShell(child: child);
        },
        routes: <RouteBase>[
          GoRoute(
            path: "/packs",
            pageBuilder: (BuildContext context, GoRouterState state) {
              return buildAppPage(
                key: state.pageKey,
                child: BlocProvider<PackListCubit>(
                  create: (_) => GetIt.instance<PackListCubit>()..load(),
                  child: const MyPacksPage(),
                ),
              );
            },
          ),
          GoRoute(
            path: "/packs/new",
            pageBuilder: (BuildContext context, GoRouterState state) {
              return buildAppPage(
                key: state.pageKey,
                child: BlocProvider<PackUploadCubit>(
                  create: (_) => GetIt.instance<PackUploadCubit>(),
                  child: const NewPackPage(),
                ),
              );
            },
          ),
          GoRoute(
            path: "/packs/:id",
            pageBuilder: (BuildContext context, GoRouterState state) {
              final String id = state.pathParameters["id"]!;
              return buildAppPage(
                key: state.pageKey,
                child: BlocProvider<PackDetailCubit>(
                  create: (_) =>
                      GetIt.instance.get<PackDetailCubit>(param1: id)..load(),
                  child: const PackDetailPage(),
                ),
              );
            },
          ),
          GoRoute(
            path: "/admin/queue",
            pageBuilder: (BuildContext context, GoRouterState state) {
              return buildAppPage(
                key: state.pageKey,
                child: BlocProvider<ReviewQueueCubit>(
                  create: (_) => GetIt.instance<ReviewQueueCubit>()..load(),
                  child: const ReviewQueuePage(),
                ),
              );
            },
          ),
          GoRoute(
            path: "/admin/queue/:id",
            pageBuilder: (BuildContext context, GoRouterState state) {
              final String id = state.pathParameters["id"]!;
              return buildAppPage(
                key: state.pageKey,
                child: BlocProvider<ReviewDetailCubit>(
                  create: (_) =>
                      GetIt.instance.get<ReviewDetailCubit>(param1: id)..load(),
                  child: const ReviewDetailPage(),
                ),
              );
            },
          ),
          GoRoute(
            path: "/admin/published",
            pageBuilder: (BuildContext context, GoRouterState state) {
              return buildAppPage(
                key: state.pageKey,
                child: BlocProvider<PublishedCatalogCubit>(
                  create: (_) => GetIt.instance<PublishedCatalogCubit>()..load(),
                  child: const PublishedCatalogPage(),
                ),
              );
            },
          ),
          GoRoute(
            path: "/admin/analytics",
            pageBuilder: (BuildContext context, GoRouterState state) {
              return buildAppPage(
                key: state.pageKey,
                child: BlocProvider<AnalyticsCubit>(
                  create: (_) => GetIt.instance<AnalyticsCubit>()..load(),
                  child: const AnalyticsPage(),
                ),
              );
            },
          ),
          GoRoute(
            path: "/keys",
            pageBuilder: (BuildContext context, GoRouterState state) {
              final bool isAdmin = authCubit.state.isAdmin;
              return buildAppPage(
                key: state.pageKey,
                child: BlocProvider<ApiKeyCubit>(
                  create: (_) =>
                      GetIt.instance<ApiKeyCubit>()..load(isAdmin: isAdmin),
                  child: const ApiKeysPage(),
                ),
              );
            },
          ),
          GoRoute(
            path: "/api-docs",
            pageBuilder: (BuildContext context, GoRouterState state) {
              return buildAppPage(
                key: state.pageKey,
                child: const ApiDocsPage(),
              );
            },
          ),
        ],
      ),
    ],
  );
}
