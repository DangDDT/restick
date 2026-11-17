import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick/src/core/restick_client.dart";

/// `X-Api-Key` from the canonical client unless [override] is given.
Map<String, String>? restickImageHeaders(
  BuildContext context, [
  Map<String, String>? override,
]) {
  if (override != null) {
    return override;
  }
  final RestickClient? client = context.read<RestickClient?>();
  return client == null ? null : <String, String>{"X-Api-Key": client.config.apiKey};
}

/// Pixel size to decode a square image shown at [logicalSize].
int restickDecodeSize(BuildContext context, double logicalSize) {
  return (logicalSize * (MediaQuery.maybeOf(context)?.devicePixelRatio ?? 1))
      .round();
}
