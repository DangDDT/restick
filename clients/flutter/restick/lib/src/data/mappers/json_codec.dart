Map<String, Object?> asJsonMap(Object? value) {
  if (value is Map<String, Object?>) {
    return value;
  }
  if (value is Map) {
    return value.map(
      (Object? key, Object? item) => MapEntry<String, Object?>(key! as String, item),
    );
  }
  return <String, Object?>{};
}

List<Object?> asJsonList(Object? value) {
  if (value is List<Object?>) {
    return value;
  }
  if (value is List) {
    return List<Object?>.from(value);
  }
  return <Object?>[];
}

List<String> asStringList(Object? value) {
  return asJsonList(value).map((Object? item) => item! as String).toList();
}
