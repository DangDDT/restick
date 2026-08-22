import "dart:typed_data";

import "package:dio/dio.dart";
import "package:material_ui/material_ui.dart";
import "package:get_it/get_it.dart";

class AuthenticatedImage extends StatefulWidget {
  const AuthenticatedImage({
    required this.url,
    super.key,
  });

  final String url;

  @override
  State<AuthenticatedImage> createState() => _AuthenticatedImageState();
}

class _AuthenticatedImageState extends State<AuthenticatedImage> {
  Uint8List? _bytes;
  Object? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void didUpdateWidget(AuthenticatedImage oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.url != widget.url) {
      _load();
    }
  }

  Future<void> _load() async {
    if (widget.url.isEmpty) {
      setState(() {
        _bytes = null;
        _error = "missing url";
      });
      return;
    }
    try {
      final Dio dio = GetIt.instance.get<Dio>(instanceName: "api");
      final Response<List<int>> response = await dio.get<List<int>>(
        widget.url,
        options: Options(responseType: ResponseType.bytes),
      );
      if (!mounted) {
        return;
      }
      setState(() {
        _bytes = Uint8List.fromList(response.data ?? <int>[]);
        _error = null;
      });
    } on Object catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _error = error;
        _bytes = null;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final ColorScheme colors = Theme.of(context).colorScheme;
    if (_error != null) {
      return ColoredBox(color: colors.surfaceContainerHighest);
    }
    if (_bytes == null) {
      return const Center(
        child: SizedBox(
          width: 18,
          height: 18,
          child: CircularProgressIndicator(strokeWidth: 2),
        ),
      );
    }
    return Image.memory(_bytes!, fit: BoxFit.contain);
  }
}
