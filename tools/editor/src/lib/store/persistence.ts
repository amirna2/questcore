/**
 * LocalStorage persistence for the editor project.
 * Saves the full EditorProject as JSON on every change,
 * restores it on page load.
 */

import type { EditorProject } from '../model/editor-types.js';

const STORAGE_KEY = 'questcore-editor-project';

export function saveProject(project: EditorProject): void {
	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(project));
	} catch {
		// Storage full or unavailable — silently skip
	}
}

export function loadProject(): EditorProject | null {
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) return null;
		return JSON.parse(raw) as EditorProject;
	} catch {
		return null;
	}
}

export function clearProject(): void {
	localStorage.removeItem(STORAGE_KEY);
}
