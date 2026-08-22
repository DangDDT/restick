import "package:equatable/equatable.dart";
import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:restick_web/core/storage/prefs_storage.dart";

class ThemeCubit extends Cubit<ThemeMode> {
  ThemeCubit({required PrefsStorage prefsStorage})
      : _prefsStorage = prefsStorage,
        super(ThemeMode.light);

  final PrefsStorage _prefsStorage;

  Future<void> restore() async {
    final String? raw = await _prefsStorage.readThemeMode();
    if (raw == "dark") {
      emit(ThemeMode.dark);
      return;
    }
    if (raw == "system") {
      emit(ThemeMode.system);
      return;
    }
    emit(ThemeMode.light);
  }

  Future<void> toggle() async {
    final ThemeMode next =
        state == ThemeMode.dark ? ThemeMode.light : ThemeMode.dark;
    emit(next);
    await _prefsStorage.writeThemeMode(next.name);
  }

  Future<void> setMode(ThemeMode mode) async {
    emit(mode);
    await _prefsStorage.writeThemeMode(mode.name);
  }
}

class LocaleState extends Equatable {
  const LocaleState(this.locale);

  final Locale locale;

  @override
  List<Object?> get props => <Object?>[locale];
}

class LocaleCubit extends Cubit<LocaleState> {
  LocaleCubit({required PrefsStorage prefsStorage})
      : _prefsStorage = prefsStorage,
        super(const LocaleState(Locale("en")));

  final PrefsStorage _prefsStorage;

  Future<void> restore() async {
    final String? code = await _prefsStorage.readLocaleCode();
    if (code == "vi") {
      emit(const LocaleState(Locale("vi")));
      return;
    }
    emit(const LocaleState(Locale("en")));
  }

  Future<void> toggle() async {
    final Locale next =
        state.locale.languageCode == "vi" ? const Locale("en") : const Locale("vi");
    emit(LocaleState(next));
    await _prefsStorage.writeLocaleCode(next.languageCode);
  }
}
