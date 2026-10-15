package com.voidclient.mixin;

import com.voidclient.Theme;
import com.voidclient.VoidClient;
import net.minecraft.client.renderer.LevelRenderer;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.injection.Constant;
import org.spongepowered.asm.mixin.injection.ModifyConstant;

@Mixin(LevelRenderer.class)
public class LevelRendererMixin {
	@ModifyConstant(method = "submitBlockOutline", constant = @Constant(intValue = -16777216))
	private int voidclient$outlineBorder(int original) {
		return VoidClient.config != null && VoidClient.config.purpleOutline ? Theme.OUTLINE_BORDER : original;
	}

	@ModifyConstant(method = "submitBlockOutline", constant = @Constant(intValue = -11010079))
	private int voidclient$outlineLine(int original) {
		return VoidClient.config != null && VoidClient.config.purpleOutline ? Theme.OUTLINE_LINE : original;
	}
}
