import "package:material_ui/material_ui.dart";
import "package:restick/src/presentation/theme/restick_brand.dart";
import "package:restick/src/presentation/theme/restick_theme.dart";

/// Eye openness (0.1 closed .. 1 open) at cycle position [v] in 0..1.
/// One blink in the last 4% of the cycle: open at 0.96, closed at 0.98,
/// open again at 1.0.
@visibleForTesting
double mascotEyeOpenness(double v) =>
    v > 0.96 ? ((v - 0.98).abs() / 0.02).clamp(0.1, 1.0) : 1.0;

enum RestickMood { happy, cheer, sad }

/// The Restick bear, repainted from `web-next/public/mascot.svg` (64x64 box).
class RestickMascot extends StatefulWidget {
  const RestickMascot({
    this.size = 64,
    this.blink = true,
    this.mood = RestickMood.happy,
    super.key,
  });

  final double size;
  final bool blink;
  final RestickMood mood;

  @override
  State<RestickMascot> createState() => _RestickMascotState();
}

class _RestickMascotState extends State<RestickMascot>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller = AnimationController(
    vsync: this,
    duration: const Duration(seconds: 4),
  );

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _sync();
  }

  @override
  void didUpdateWidget(RestickMascot old) {
    super.didUpdateWidget(old);
    _sync();
  }

  void _sync() {
    final bool run = widget.blink && !MediaQuery.disableAnimationsOf(context);
    if (!run) {
      _controller.stop();
    } else if (!_controller.isAnimating) {
      _controller.repeat();
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final RestickTheme theme = RestickTheme.of(context);
    return RepaintBoundary(
      child: AnimatedBuilder(
        animation: _controller,
        builder: (BuildContext context, Widget? _) {
          final double open = mascotEyeOpenness(_controller.value);
          return CustomPaint(
            size: Size.square(widget.size),
            painter: _MascotPainter(widget.mood, open, theme),
          );
        },
      ),
    );
  }
}

class _MascotPainter extends CustomPainter {
  _MascotPainter(this.mood, this.eyeOpen, this.theme);

  final RestickMood mood;
  final double eyeOpen;
  final RestickTheme theme;

  @override
  void paint(Canvas canvas, Size size) {
    canvas.scale(size.width / 64, size.height / 64);
    Paint fill(Color c) => Paint()..color = c;
    void circle(double x, double y, double r, Color c) =>
        canvas.drawCircle(Offset(x, y), r, fill(c));
    void oval(double cx, double cy, double rx, double ry, Paint p) =>
        canvas.drawOval(
          Rect.fromCenter(
            center: Offset(cx, cy),
            width: rx * 2,
            height: ry * 2,
          ),
          p,
        );

    const Color white = Color(0xFFFFFFFF);
    circle(15, 16, 9.5, theme.brand);
    circle(49, 16, 9.5, theme.brand);
    circle(15.5, 16.5, 5, theme.brandSoft);
    circle(48.5, 16.5, 5, theme.brandSoft);
    oval(32, 36, 25.5, 22.5, fill(theme.brand));
    oval(32, 43.5, 11, 8.5, fill(white));

    final Paint stroke =
        Paint()
          ..color = theme.ink
          ..style = PaintingStyle.stroke
          ..strokeWidth = 1.9
          ..strokeCap = StrokeCap.round;

    if (mood == RestickMood.cheer) {
      // Happy closed arcs ^ ^ instead of round eyes.
      for (final double x in <double>[22, 42]) {
        final Path p =
            Path()
              ..moveTo(x - 3.4, 34)
              ..quadraticBezierTo(x, 28.5, x + 3.4, 34);
        canvas.drawPath(p, stroke);
      }
    } else {
      final double ry = 3.4 * eyeOpen;
      for (final double x in <double>[22, 42]) {
        oval(x, 33, 3.4, ry, fill(theme.ink));
        if (eyeOpen > 0.5) {
          circle(x + 1.2, 31.8, 1.1, white);
        }
      }
      if (mood == RestickMood.sad) {
        // Slanted brows.
        canvas
          ..drawLine(const Offset(18.5, 27.5), const Offset(25, 29), stroke)
          ..drawLine(const Offset(45.5, 27.5), const Offset(39, 29), stroke);
      }
    }

    final Paint cheek = fill(RestickBrand.cheekPink.withValues(alpha: .8));
    oval(14, 41, 4.6, 3.2, cheek);
    oval(50, 41, 4.6, 3.2, cheek);
    oval(32, 40.6, 3, 2.2, fill(theme.ink));

    final Path mouth = Path();
    switch (mood) {
      case RestickMood.happy:
        mouth
          ..moveTo(27.8, 45.2)
          ..cubicTo(29.7, 47.2, 34.3, 47.2, 36.2, 45.2);
      case RestickMood.cheer:
        // Open smile.
        mouth
          ..moveTo(27, 44.5)
          ..quadraticBezierTo(32, 52, 37, 44.5)
          ..close();
        canvas.drawPath(mouth, fill(RestickBrand.deepPink));
        return;
      case RestickMood.sad:
        mouth
          ..moveTo(27.8, 47.4)
          ..cubicTo(29.7, 45.2, 34.3, 45.2, 36.2, 47.4);
    }
    canvas.drawPath(mouth, stroke);
  }

  @override
  bool shouldRepaint(_MascotPainter old) =>
      old.mood != mood || old.eyeOpen != eyeOpen || old.theme != theme;
}
