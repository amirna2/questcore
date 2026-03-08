/**
 * Known verbs recognized by the QuestCore engine, with aliases.
 * Used by the VerbPicker component and validation warnings.
 */

export interface VerbEntry {
	verb: string;
	aliases: string[];
}

export const KNOWN_VERBS: VerbEntry[] = [
	{ verb: 'look', aliases: ['examine', 'inspect', 'l', 'x'] },
	{ verb: 'take', aliases: ['get', 'grab', 'pick up'] },
	{ verb: 'drop', aliases: ['put down', 'discard'] },
	{ verb: 'use', aliases: ['apply'] },
	{ verb: 'open', aliases: [] },
	{ verb: 'close', aliases: ['shut'] },
	{ verb: 'talk', aliases: ['speak', 'chat'] },
	{ verb: 'give', aliases: ['hand', 'offer'] },
	{ verb: 'push', aliases: ['shove', 'press'] },
	{ verb: 'pull', aliases: ['yank', 'tug'] },
	{ verb: 'read', aliases: [] },
	{ verb: 'attack', aliases: ['hit', 'fight', 'strike'] },
	{ verb: 'inventory', aliases: ['i', 'inv'] },
	{ verb: 'help', aliases: ['h', '?'] }
];

export const ALL_VERB_NAMES = KNOWN_VERBS.map((v) => v.verb);
