import "package:material_ui/material_ui.dart";
import "package:flutter_bloc/flutter_bloc.dart";
import "package:go_router/go_router.dart";
import "package:restick_web/core/error/error_cubit.dart";
import "package:restick_web/core/error/failure.dart";
import "package:restick_web/features/auth/presentation/cubit/auth_cubit.dart";
import "package:restick_web/l10n/app_localizations.dart";

class RegisterPage extends StatefulWidget {
  const RegisterPage({super.key});

  @override
  State<RegisterPage> createState() => _RegisterPageState();
}

class _RegisterPageState extends State<RegisterPage> {
  final TextEditingController _email = TextEditingController();
  final TextEditingController _password = TextEditingController();
  final TextEditingController _displayName = TextEditingController();

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    _displayName.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    try {
      await context.read<AuthCubit>().register(
            email: _email.text.trim(),
            password: _password.text,
            displayName: _displayName.text.trim(),
          );
      if (mounted) {
        context.go("/packs");
      }
    } on Failure catch (failure) {
      if (mounted) {
        context.read<ErrorCubit>().showFailure(failure.message);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final AppLocalizations l10n = AppLocalizations.of(context);
    final AuthState auth = context.watch<AuthCubit>().state;
    final ColorScheme colors = Theme.of(context).colorScheme;
    return Scaffold(
      backgroundColor: colors.surfaceContainerLowest,
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 440),
          child: Card(
            child: Padding(
              padding: const EdgeInsets.all(28),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                spacing: 14,
                children: [
                  Text(
                    l10n.signUp,
                    style: Theme.of(context).textTheme.headlineMedium,
                  ),
                  TextField(
                    controller: _displayName,
                    decoration: InputDecoration(labelText: l10n.displayName),
                  ),
                  TextField(
                    controller: _email,
                    decoration: InputDecoration(labelText: l10n.email),
                  ),
                  TextField(
                    controller: _password,
                    obscureText: true,
                    decoration: InputDecoration(labelText: l10n.password),
                  ),
                  if (auth.errorMessage != null)
                    Text(
                      auth.errorMessage!,
                      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                            color: colors.error,
                          ),
                    ),
                  SizedBox(
                    width: double.infinity,
                    child: FilledButton(
                      onPressed: auth.isBusy ? null : _submit,
                      child: Text(
                        auth.isBusy ? l10n.creatingAccount : l10n.signUp,
                      ),
                    ),
                  ),
                  TextButton(
                    onPressed: () => context.go("/login"),
                    child: Text(l10n.signIn),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
