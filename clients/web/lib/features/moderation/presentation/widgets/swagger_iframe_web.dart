import "dart:ui_web" as ui_web;

import "package:material_ui/material_ui.dart";
import "package:web/web.dart" as web;

class SwaggerIframe extends StatefulWidget {
  const SwaggerIframe({required this.src, super.key});

  final String src;

  @override
  State<SwaggerIframe> createState() => _SwaggerIframeState();
}

class _SwaggerIframeState extends State<SwaggerIframe> {
  static const String _viewType = "restick-swagger-docs";
  static bool _registered = false;
  static String _src = "";

  @override
  void initState() {
    super.initState();
    _src = widget.src;
    if (!_registered) {
      ui_web.platformViewRegistry.registerViewFactory(_viewType, (int _) {
        final web.HTMLIFrameElement iframe = web.HTMLIFrameElement()
          ..src = _src
          ..style.border = "none"
          ..style.width = "100%"
          ..style.height = "100%"
          ..setAttribute("allow", "clipboard-read; clipboard-write")
          ..setAttribute("title", "Restick API docs");
        return iframe;
      });
      _registered = true;
    }
  }

  @override
  Widget build(BuildContext context) {
    return const HtmlElementView(viewType: _viewType);
  }
}
