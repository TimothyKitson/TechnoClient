package com.voidclient;

public class Zoom {
	private static final double MIN_FACTOR = 1.5;
	private static final double MAX_FACTOR = 50.0;

	private static boolean active;
	private static double factor = 4.0;
	private static double current = 1.0;
	private static long lastNanos = System.nanoTime();

	public static void setActive(boolean value) {
		if (value && !active) {
			factor = VoidClient.config.zoomFactor;
		}
		active = value;
	}

	public static boolean isActive() {
		return active;
	}

	public static void scroll(double amount) {
		factor = Math.clamp(factor * Math.pow(1.2, amount), MIN_FACTOR, MAX_FACTOR);
	}

	public static double divisor() {
		long now = System.nanoTime();
		double seconds = Math.min((now - lastNanos) / 1_000_000_000.0, 0.25);
		lastNanos = now;
		double target = active ? factor : 1.0;
		if (!VoidClient.config.smoothZoom) {
			current = target;
		} else {
			current += (target - current) * (1.0 - Math.exp(-seconds * 18.0));
			if (Math.abs(target - current) < 0.001) {
				current = target;
			}
		}
		return current;
	}

	public static double sensitivityScale() {
		return active ? 1.0 / factor : 1.0;
	}
}
