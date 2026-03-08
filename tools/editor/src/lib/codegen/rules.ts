/**
 * Generate rules.lua from Rule[] and EventHandler[].
 */

import type { EventHandler, Rule, WhenClause } from '../model/export-types.js';
import { luaString, luaValue } from './lua-utils.js';
import { generateCondition } from './conditions.js';
import { generateEffect } from './effects.js';

export function generateRules(rules: Rule[]): string {
	return rules.map(generateRule).join('\n');
}

export function generateEvents(events: EventHandler[]): string {
	return events.map(generateEvent).join('\n');
}

function generateRule(rule: Rule): string {
	const lines: string[] = [];
	const whenStr = generateWhen(rule.when);
	const hasConditions = rule.conditions.length > 0;

	lines.push(`Rule(${luaString(rule.id)},`);
	lines.push(`    ${whenStr},`);

	if (hasConditions) {
		const conds = rule.conditions.map(generateCondition).join(', ');
		lines.push(`    { ${conds} },`);
	}

	// Then block
	if (rule.effects.length === 1) {
		lines.push(`    Then {`);
		lines.push(`        ${generateEffect(rule.effects[0])}`);
		lines.push('    }');
	} else {
		lines.push('    Then {');
		for (const effect of rule.effects) {
			lines.push(`        ${generateEffect(effect)},`);
		}
		lines.push('    }');
	}

	lines.push(')');
	return lines.join('\n') + '\n';
}

function generateWhen(when: WhenClause): string {
	const fields: string[] = [];
	fields.push(`verb = ${luaString(when.verb)}`);

	if (when.object) {
		fields.push(`object = ${luaString(when.object)}`);
	}
	if (when.target) {
		fields.push(`target = ${luaString(when.target)}`);
	}
	if (when.objectKind) {
		fields.push(`object_kind = ${luaString(when.objectKind)}`);
	}
	if (when.priority !== undefined) {
		fields.push(`priority = ${when.priority}`);
	}
	if (when.objectProp) {
		for (const [k, v] of Object.entries(when.objectProp)) {
			fields.push(`object_prop_${k} = ${luaValue(v)}`);
		}
	}
	if (when.targetProp) {
		for (const [k, v] of Object.entries(when.targetProp)) {
			fields.push(`target_prop_${k} = ${luaValue(v)}`);
		}
	}

	return `When { ${fields.join(', ')} }`;
}

function generateEvent(event: EventHandler): string {
	const lines: string[] = [];
	lines.push(`On(${luaString(event.event)}, {`);

	if (event.conditions.length > 0) {
		lines.push('    conditions = {');
		for (const cond of event.conditions) {
			lines.push(`        ${generateCondition(cond)},`);
		}
		lines.push('    },');
	}

	lines.push('    effects = {');
	for (const effect of event.effects) {
		lines.push(`        ${generateEffect(effect)},`);
	}
	lines.push('    }');

	lines.push('})');
	return lines.join('\n') + '\n';
}
