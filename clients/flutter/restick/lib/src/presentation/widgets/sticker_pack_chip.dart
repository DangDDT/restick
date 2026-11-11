import "package:auto_size_text/auto_size_text.dart";
import "package:flutter/material.dart";
import "package:restick/src/domain/entities/sticker_pack.dart";

class StickerPackChip extends StatelessWidget {
  const StickerPackChip({
    required this.pack,
    required this.selected,
    required this.onSelected,
    super.key,
  });

  final StickerPack pack;
  final bool selected;
  final ValueChanged<StickerPack> onSelected;

  @override
  Widget build(BuildContext context) {
    return FilterChip(
      key: ValueKey('sticker.pack.chip.${pack.id}'),
      label: SizedBox(
        width: 96,
        child: AutoSizeText(
          pack.name,
          maxLines: 1,
          textAlign: TextAlign.center,
        ),
      ),
      selected: selected,
      onSelected: (_) => onSelected(pack),
    );
  }
}
