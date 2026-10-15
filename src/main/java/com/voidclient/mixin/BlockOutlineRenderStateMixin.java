package com.voidclient.mixin;

import com.voidclient.VoidClient;
import net.minecraft.client.renderer.state.level.BlockOutlineRenderState;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfoReturnable;

@Mixin(BlockOutlineRenderState.class)
public class BlockOutlineRenderStateMixin {
	@Inject(method = "highContrast", at = @At("HEAD"), cancellable = true)
	private void voidclient$forcePurpleOutline(CallbackInfoReturnable<Boolean> cir) {
		if (VoidClient.config != null && VoidClient.config.purpleOutline) {
			cir.setReturnValue(true);
		}
	}
}
