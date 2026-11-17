-- Marketplace card colour. theme_color is the admin override, theme_color_suggested
-- the colour derived from the pack's stickers; both are '' (unset) or #RRGGBB
-- uppercase, and the effective colour is the override when set.

ALTER TABLE packs ADD COLUMN theme_color TEXT NOT NULL DEFAULT ''
    CHECK (theme_color = '' OR theme_color GLOB '#[0-9A-F][0-9A-F][0-9A-F][0-9A-F][0-9A-F][0-9A-F]');
ALTER TABLE packs ADD COLUMN theme_color_suggested TEXT NOT NULL DEFAULT ''
    CHECK (theme_color_suggested = '' OR theme_color_suggested GLOB '#[0-9A-F][0-9A-F][0-9A-F][0-9A-F][0-9A-F][0-9A-F]');
