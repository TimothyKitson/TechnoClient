package com.voidclient.gui;

import com.voidclient.VoidClient;
import com.voidclient.VoidConfig;
import com.voidclient.Theme;
import net.fabricmc.fabric.api.client.rendering.v1.hud.HudElement;
import net.minecraft.client.DeltaTracker;
import net.minecraft.client.KeyMapping;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.Font;
import net.minecraft.client.gui.GuiGraphicsExtractor;
import net.minecraft.client.multiplayer.ClientPacketListener;
import net.minecraft.client.multiplayer.PlayerInfo;
import net.minecraft.client.player.LocalPlayer;

public class VoidHud implements HudElement {
	private static final int MARGIN = 4;
	private static final int LINE_HEIGHT = 12;
	private static final int KEY_SIZE = 22;
	private static final int GAP = 2;

	@Override
	public void extractRenderState(GuiGraphicsExtractor graphics, DeltaTracker deltaTracker) {
		Minecraft minecraft = Minecraft.getInstance();
		LocalPlayer player = minecraft.player;
		if (player == null || minecraft.getDebugOverlay().showDebugScreen()) {
			return;
		}
		VoidConfig config = VoidClient.config;
		Font font = minecraft.font;
		int y = MARGIN;
		if (config.showFps) {
			y = line(graphics, font, y, "FPS", Integer.toString(minecraft.getFps()));
		}
		if (config.showCps) {
			y = line(graphics, font, y, "CPS", VoidClient.LEFT_CLICKS.perSecond() + " | " + VoidClient.RIGHT_CLICKS.perSecond());
		}
		if (config.showPing) {
			int ping = ping(minecraft, player);
			if (ping >= 0) {
				y = line(graphics, font, y, "Ping", ping + " ms");
			}
		}
		if (config.showCoords && !minecraft.showOnlyReducedInfo()) {
			y = line(graphics, font, y, "XYZ", player.getBlockX() + " " + player.getBlockY() + " " + player.getBlockZ());
			y = line(graphics, font, y, "Facing", capitalize(player.getDirection().getName()));
		}
		if (config.showKeystrokes) {
			keystrokes(graphics, font, minecraft, y + GAP);
		}
	}

	private static int line(GuiGraphicsExtractor graphics, Font font, int y, String label, String value) {
		int labelWidth = font.width(label);
		int width = 3 + labelWidth + 4 + font.width(value) + 4;
		graphics.fill(MARGIN, y, MARGIN + width, y + LINE_HEIGHT, Theme.BLACK_PANEL);
		graphics.fill(MARGIN, y, MARGIN + 1, y + LINE_HEIGHT, Theme.PURPLE);
		graphics.text(font, label, MARGIN + 4, y + 2, Theme.PURPLE_BRIGHT, false);
		graphics.text(font, value, MARGIN + 4 + labelWidth + 4, y + 2, Theme.TEXT, false);
		return y + LINE_HEIGHT + 1;
	}

	private static void keystrokes(GuiGraphicsExtractor graphics, Font font, Minecraft minecraft, int top) {
		int step = KEY_SIZE + GAP;
		int fullWidth = KEY_SIZE * 3 + GAP * 2;
		key(graphics, font, MARGIN + step, top, KEY_SIZE, KEY_SIZE, label(font, minecraft.options.keyUp), minecraft.options.keyUp.isDown());
		key(graphics, font, MARGIN, top + step, KEY_SIZE, KEY_SIZE, label(font, minecraft.options.keyLeft), minecraft.options.keyLeft.isDown());
		key(graphics, font, MARGIN + step, top + step, KEY_SIZE, KEY_SIZE, label(font, minecraft.options.keyDown), minecraft.options.keyDown.isDown());
		key(graphics, font, MARGIN + step * 2, top + step, KEY_SIZE, KEY_SIZE, label(font, minecraft.options.keyRight), minecraft.options.keyRight.isDown());
		int mouseWidth = (fullWidth - GAP) / 2;
		int mouseTop = top + step * 2;
		key(graphics, font, MARGIN, mouseTop, mouseWidth, KEY_SIZE, "LMB", minecraft.options.keyAttack.isDown());
		key(graphics, font, MARGIN + mouseWidth + GAP, mouseTop, fullWidth - mouseWidth - GAP, KEY_SIZE, "RMB", minecraft.options.keyUse.isDown());
		int spaceTop = mouseTop + step;
		boolean jump = minecraft.options.keyJump.isDown();
		graphics.fill(MARGIN, spaceTop, MARGIN + fullWidth, spaceTop + 12, jump ? Theme.PURPLE_PRESSED : Theme.BLACK_PANEL);
		graphics.fill(MARGIN + fullWidth / 2 - 12, spaceTop + 5, MARGIN + fullWidth / 2 + 12, spaceTop + 7, jump ? Theme.TEXT : Theme.PURPLE_BRIGHT);
	}

	private static void key(GuiGraphicsExtractor graphics, Font font, int x, int y, int width, int height, String text, boolean down) {
		graphics.fill(x, y, x + width, y + height, down ? Theme.PURPLE_PRESSED : Theme.BLACK_PANEL);
		if (!down) {
			graphics.fill(x, y + height - 1, x + width, y + height, Theme.PURPLE_DEEP);
		}
		graphics.centeredText(font, text, x + width / 2, y + (height - font.lineHeight) / 2 + 1, Theme.TEXT);
	}

	private static String label(Font font, KeyMapping mapping) {
		String name = mapping.getTranslatedKeyMessage().getString();
		if (font.width(name) > KEY_SIZE - 4) {
			return name.substring(0, 1).toUpperCase();
		}
		return name;
	}

	private static int ping(Minecraft minecraft, LocalPlayer player) {
		if (minecraft.hasSingleplayerServer()) {
			return -1;
		}
		ClientPacketListener connection = minecraft.getConnection();
		if (connection == null) {
			return -1;
		}
		PlayerInfo info = connection.getPlayerInfo(player.getUUID());
		return info == null ? -1 : info.getLatency();
	}

	private static String capitalize(String text) {
		return text.isEmpty() ? text : Character.toUpperCase(text.charAt(0)) + text.substring(1);
	}
}
