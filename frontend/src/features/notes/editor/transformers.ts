import {
	BOLD_ITALIC_STAR,
	BOLD_ITALIC_UNDERSCORE,
	BOLD_STAR,
	BOLD_UNDERSCORE,
	CHECK_LIST,
	CODE,
	type ElementTransformer,
	INLINE_CODE,
	ITALIC_STAR,
	ITALIC_UNDERSCORE,
	LINK,
	ORDERED_LIST,
	QUOTE,
	STRIKETHROUGH,
	type TextMatchTransformer,
	type Transformer,
	UNORDERED_LIST,
} from "@lexical/markdown";
import {
	$createHorizontalRuleNode,
	HorizontalRuleNode,
} from "@lexical/react/LexicalHorizontalRuleNode";
import { $createHeadingNode, HeadingNode } from "@lexical/rich-text";
import { isLinkUrl } from "../document/types";

/** `# `, `## ` and `### `: the document format has three heading levels. */
const HEADING: ElementTransformer = {
	dependencies: [HeadingNode],
	export: () => null,
	regExp: /^(#{1,3})\s/,
	replace: (parentNode, children, match) => {
		const node = $createHeadingNode(
			`h${match[1].length}` as "h1" | "h2" | "h3",
		);
		node.append(...children);
		parentNode.replace(node);
		node.select(0, 0);
	},
	type: "element",
};

/** `---`, `***` or `___` alone on a line, followed by a space or Enter. */
const DIVIDER: ElementTransformer = {
	dependencies: [HorizontalRuleNode],
	export: () => null,
	regExp: /^(---|\*\*\*|___)\s?$/,
	replace: (parentNode, _children, _match, isImport) => {
		const line = $createHorizontalRuleNode();
		if (isImport || parentNode.getNextSibling() !== null) {
			parentNode.replace(line);
		} else {
			parentNode.insertBefore(line);
		}
		line.selectNext();
	},
	triggerOnEnter: true,
	type: "element",
};

/** `[text](url)`, only for http, https and mailto URLs. */
const SAFE_LINK: TextMatchTransformer = {
	...LINK,
	replace: (node, match) => {
		const url = match[2] ?? match[3];
		if (!url || !isLinkUrl(url)) return;
		return LINK.replace?.(node, match);
	},
};

/**
 * The Markdown shortcuts of the note editor (see the note-document spec).
 * Task lists come before bullet lists so `- [ ] ` makes a task.
 */
export const noteTransformers: Transformer[] = [
	HEADING,
	QUOTE,
	CHECK_LIST,
	UNORDERED_LIST,
	ORDERED_LIST,
	DIVIDER,
	CODE,
	BOLD_ITALIC_STAR,
	BOLD_ITALIC_UNDERSCORE,
	BOLD_STAR,
	BOLD_UNDERSCORE,
	ITALIC_STAR,
	ITALIC_UNDERSCORE,
	STRIKETHROUGH,
	INLINE_CODE,
	SAFE_LINK,
];
