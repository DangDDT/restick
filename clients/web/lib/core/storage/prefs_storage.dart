import "package:shared_preferences/shared_preferences.dart";

class PrefsStorage {
  PrefsStorage({required SharedPreferences preferences})
      : _preferences = preferences;

  final SharedPreferences _preferences;

  static const String _themeKey = "theme_mode";
  static const String _localeKey = "locale_code";

  Future<String?> readThemeMode() async => _preferences.getString(_themeKey);

  Future<void> writeThemeMode(String mode) async {
    await _preferences.setString(_themeKey, mode);
  }

  Future<String?> readLocaleCode() async => _preferences.getString(_localeKey);

  Future<void> writeLocaleCode(String code) async {
    await _preferences.setString(_localeKey, code);
  }
}
