String rewritePublicUrl(String advertised, Uri apiBase) {
  if (advertised.isEmpty) {
    return advertised;
  }
  final Uri parsed = Uri.parse(advertised);
  if (!_isLoopbackHost(parsed.host)) {
    return advertised;
  }
  return parsed.replace(
    scheme: apiBase.scheme,
    host: apiBase.host,
    port: apiBase.hasPort ? apiBase.port : (apiBase.scheme == "https" ? 443 : 80),
  ).toString();
}

bool _isLoopbackHost(String host) {
  final String normalized = host.toLowerCase();
  return normalized == "localhost" ||
      normalized == "127.0.0.1" ||
      normalized == "::1" ||
      normalized == "0.0.0.0";
}
