import "package:material_ui/material_ui.dart";
import "package:get_it/get_it.dart";
import "package:restick_web/core/config/env_config.dart";
import "package:restick_web/features/moderation/presentation/widgets/swagger_iframe.dart";
import "package:restick_web/l10n/app_localizations.dart";

class ApiDocsPage extends StatelessWidget {
  const ApiDocsPage({super.key});

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final EnvConfig config = GetIt.instance<EnvConfig>();
    final String docsUrl = "${config.apiBaseUrl}/docs";
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        spacing: 12,
        children: [
          Text(
            l10n.titleApiDocs,
            style: Theme.of(context).textTheme.headlineSmall,
          ),
          Text(
            l10n.descApiDocs,
            style: Theme.of(context).textTheme.bodyMedium,
          ),
          Expanded(
            child: ClipRRect(
              borderRadius: BorderRadius.circular(12),
              child: SwaggerIframe(src: docsUrl),
            ),
          ),
        ],
      ),
    );
  }
}
