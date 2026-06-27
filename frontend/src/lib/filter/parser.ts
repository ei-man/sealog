import type { AstNode } from './ast';
import { tokenize, type Token, type TokenType } from './lexer';

export class ParseError extends Error {
  constructor(
    message: string,
    public position: number
  ) {
    super(message);
    this.name = 'ParseError';
  }
}

const VALUE_TOKENS = new Set<TokenType>(['identifier', 'string', 'and', 'or', 'not']);

class Parser {
  private index = 0;

  constructor(private tokens: Token[]) {}

  parse(): AstNode {
    const expr = this.parseOr();
    this.expect('eof', 'Unexpected token after query');
    return expr;
  }

  private parseOr(): AstNode {
    let left = this.parseAnd();
    while (this.match('or')) {
      const right = this.parseAnd();
      left = { type: 'or', left, right };
    }
    return left;
  }

  private parseAnd(): AstNode {
    let left = this.parseUnary();
    while (true) {
      if (this.match('and')) {
        const right = this.parseUnary();
        left = { type: 'and', left, right };
        continue;
      }
      if (this.startsUnary(this.peek().type)) {
        const right = this.parseUnary();
        left = { type: 'and', left, right };
        continue;
      }
      return left;
    }
  }

  private parseUnary(): AstNode {
    if (this.match('not')) {
      return { type: 'not', expr: this.parseUnary() };
    }
    return this.parsePrimary();
  }

  private parsePrimary(): AstNode {
    if (this.match('leftParen')) {
      const expr = this.parseOr();
      this.expect('rightParen', 'Expected closing parenthesis');
      return expr;
    }
    return this.parseCondition();
  }

  private parseCondition(): AstNode {
    const field = this.expect('identifier', 'Expected field name');
    let operator: '=' | '~=';
    if (this.match('regexEquals')) {
      operator = '~=';
    } else {
      this.expect('equals', 'Expected = or ~=');
      operator = '=';
    }

    const token = this.peek();
    if (!VALUE_TOKENS.has(token.type)) {
      throw new ParseError('Expected filter value', token.position);
    }
    this.advance();

    if (operator === '~=') {
      try {
        return {
          type: 'condition',
          field: field.value,
          operator,
          value: token.value,
          regex: new RegExp(token.value, 'i')
        };
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Invalid regular expression';
        throw new ParseError(message, token.position);
      }
    }

    return {
      type: 'condition',
      field: field.value,
      operator,
      value: token.value
    };
  }

  private startsUnary(type: TokenType): boolean {
    return type === 'identifier' || type === 'not' || type === 'leftParen';
  }

  private match(type: TokenType): boolean {
    if (this.peek().type !== type) return false;
    this.advance();
    return true;
  }

  private expect(type: TokenType, message: string): Token {
    const token = this.peek();
    if (token.type !== type) throw new ParseError(message, token.position);
    return this.advance();
  }

  private peek(): Token {
    return this.tokens[this.index];
  }

  private advance(): Token {
    return this.tokens[this.index++];
  }
}

export function parseQuery(input: string): AstNode {
  const tokens = tokenize(input);
  if (tokens[0]?.type === 'eof') throw new ParseError('Enter a filter expression', 0);
  return new Parser(tokens).parse();
}
