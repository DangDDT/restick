import "package:equatable/equatable.dart";
import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";

enum ToastKind { success, danger, warning, info }

class ToastMessage extends Equatable {
  const ToastMessage({
    required this.id,
    required this.kind,
    required this.title,
    required this.body,
  });

  final String id;
  final ToastKind kind;
  final String title;
  final String body;

  @override
  List<Object?> get props => <Object?>[id, kind, title, body];
}

class ErrorCubit extends Cubit<List<ToastMessage>> {
  ErrorCubit() : super(const <ToastMessage>[]);

  void show({
    required ToastKind kind,
    required String title,
    required String body,
  }) {
    final ToastMessage message = ToastMessage(
      id: UniqueKey().toString(),
      kind: kind,
      title: title,
      body: body,
    );
    emit(<ToastMessage>[...state, message]);
  }

  void showFailure(String message) {
    show(kind: ToastKind.danger, title: "Error", body: message);
  }

  void dismiss(String id) {
    emit(state.where((ToastMessage item) => item.id != id).toList());
  }
}
