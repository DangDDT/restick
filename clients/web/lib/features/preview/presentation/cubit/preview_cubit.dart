import "package:equatable/equatable.dart";
import "package:flutter_bloc/flutter_bloc.dart";

class PreviewMessage extends Equatable {
  const PreviewMessage({
    required this.id,
    required this.isSent,
    this.text,
    this.stickerUrl,
  });

  final String id;
  final bool isSent;
  final String? text;
  final String? stickerUrl;

  @override
  List<Object?> get props => <Object?>[id, isSent, text, stickerUrl];
}

class PreviewState extends Equatable {
  const PreviewState({this.messages = const <PreviewMessage>[]});

  final List<PreviewMessage> messages;

  @override
  List<Object?> get props => <Object?>[messages];
}

class PreviewCubit extends Cubit<PreviewState> {
  PreviewCubit()
      : super(
          const PreviewState(
            messages: <PreviewMessage>[
              PreviewMessage(
                id: "intro",
                isSent: false,
                text: "intro",
              ),
              PreviewMessage(
                id: "prompt",
                isSent: true,
                text: "prompt",
              ),
            ],
          ),
        );

  void sendSticker(String url) {
    final PreviewMessage message = PreviewMessage(
      id: DateTime.now().microsecondsSinceEpoch.toString(),
      isSent: true,
      stickerUrl: url,
    );
    emit(PreviewState(messages: <PreviewMessage>[...state.messages, message]));
  }

  void clearStickers() {
    emit(
      PreviewState(
        messages: state.messages
            .where((PreviewMessage m) => m.stickerUrl == null)
            .toList(),
      ),
    );
  }
}
