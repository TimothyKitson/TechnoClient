package com.voidclient;

import java.util.ArrayDeque;

public class ClickCounter {
	private static final long WINDOW_NANOS = 1_000_000_000L;
	private final ArrayDeque<Long> clicks = new ArrayDeque<>();

	public void click() {
		clicks.addLast(System.nanoTime());
	}

	public int perSecond() {
		long cutoff = System.nanoTime() - WINDOW_NANOS;
		while (!clicks.isEmpty() && clicks.peekFirst() < cutoff) {
			clicks.removeFirst();
		}
		return clicks.size();
	}
}
