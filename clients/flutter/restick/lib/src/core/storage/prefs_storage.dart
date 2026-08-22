import "package:shared_preferences/shared_preferences.dart";

class RestickPrefsStorage {
  RestickPrefsStorage({required SharedPreferences preferences})
      : _preferences = preferences;

  static const String selectedPackKey = "restick.selected_pack_id";

  final SharedPreferences _preferences;

  Future<void> writeSelectedPackId(String packId) {
    return _preferences.setString(selectedPackKey, packId);
  }

  String? readSelectedPackId() {
    return _preferences.getString(selectedPackKey);
  }
}
