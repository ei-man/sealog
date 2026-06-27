import type { AstNode } from './ast';
import type { LogLine } from '../types';
import { valueToText } from '../utils';

export function evaluateFilter(ast: AstNode, line: LogLine): boolean {
  switch (ast.type) {
    case 'and':
      return evaluateFilter(ast.left, line) && evaluateFilter(ast.right, line);
    case 'or':
      return evaluateFilter(ast.left, line) || evaluateFilter(ast.right, line);
    case 'not':
      return !evaluateFilter(ast.expr, line);
    case 'condition': {
      if (!Object.prototype.hasOwnProperty.call(line.fields, ast.field)) return false;
      const text = valueToText(line.fields[ast.field]);
      if (ast.operator === '~=') return ast.regex?.test(text) ?? false;
      return text.toLocaleLowerCase() === ast.value.toLocaleLowerCase();
    }
  }
}
