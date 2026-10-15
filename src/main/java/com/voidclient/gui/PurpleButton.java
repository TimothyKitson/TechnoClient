package com.voidclient.gui;

import com.voidclient.Theme;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.Font;
import net.minecraft.client.gui.GuiGraphicsExtractor;
import net.minecraft.client.gui.components.AbstractButton;
import net.minecraft.client.gui.narration.NarrationElementOutput;
import net.minecraft.client.input.InputWithModifiers;
import net.minecraft.network.chat.Component;

import java.util.function.Consumer;

public class PurpleButton extends AbstractButton {
	private final Consumer<PurpleButton> action;

	public PurpleButton(int x, int y, int width, int height, Component message, Consumer<PurpleButton> action) {
		super(x, y, width, height, message);
		this.action = action;
	}

	@Override
	public void onPress(InputWithModifiers input) {
		action.accept(this);
	}

	@Override
	protected void extractContents(GuiGraphicsExtractor graphics, int mouseX, int mouseY, float partialTick) {
		int x = getX();
		int y = getY();
		int right = x + getWidth();
		int bottom = y + getHeight();
		boolean highlighted = isActive() && isHoveredOrFocused();
		graphics.fill(x, y, right, bottom, highlighted ? Theme.PURPLE : Theme.BLACK_SOLID);
		int border = highlighted ? Theme.PURPLE_BRIGHT : Theme.PURPLE_DEEP;
		graphics.fill(x, y, right, y + 1, border);
		graphics.fill(x, bottom - 1, right, bottom, border);
		graphics.fill(x, y, x + 1, bottom, border);
		graphics.fill(right - 1, y, right, bottom, border);
		Font font = Minecraft.getInstance().font;
		int color = isActive() ? Theme.TEXT : Theme.TEXT_MUTED;
		graphics.centeredText(font, getMessage(), x + getWidth() / 2, y + (getHeight() - font.lineHeight) / 2 + 1, color);
	}

	@Override
	protected void updateWidgetNarration(NarrationElementOutput output) {
		defaultButtonNarrationText(output);
	}
}
