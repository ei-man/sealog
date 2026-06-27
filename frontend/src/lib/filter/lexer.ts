export type TokenType = 'identifier' | 'string' | 'equals' | 'regexEquals' | 'and' | 'or' | 'not' | 'leftParen' | 'rightParen' | 'eof';

export type Token = {
  type: TokenType;
  value: string;
  position: number;
};

const KEYWORDS: Record<string, TokenType> = {
  AND: 'and',
  OR: 'or',
  NOT: 'not'
};

export class LexerError extends Error {
  constructor(
    message: string,
    public position: number
  ) {
    super(message);
    this.name = 'LexerError';
  }
}

export function tokenize(input: string): Token[] {
  const tokens: Token[] = [];
  let index = 0;

  while (index < input.length) {
    const char = input[index];

    if (/\s/.test(char)) {
      index++;
      continue;
    }

    if (char === '(') {
      tokens.push({ type: 'leftParen', value: char, position: index++ });
      continue;
    }

    if (char === ')') {
      tokens.push({ type: 'rightParen', value: char, position: index++ });
      continue;
    }

    if (char === '~' && input[index + 1] === '=') {
      tokens.push({ type: 'regexEquals', value: '~=', position: index });
      index += 2;
      continue;
    }

    if (char === '=') {
      tokens.push({ type: 'equals', value: '=', position: index++ });
      continue;
    }

    if (char === '"') {
      const start = index;
      index++;
      let value = '';
      while (index < input.length) {
        const current = input[index];
        if (current === '\\') {
          const next = input[index + 1];
          if (next === undefined) throw new LexerError('Unterminated escape sequence', index);
          value += next;
          index += 2;
          continue;
        }
        if (current === '"') {
          index++;
          tokens.push({ type: 'string', value, position: start });
          value = '';
          break;
        }
        value += current;
        index++;
      }
      if (index > input.length || input[index - 1] !== '"') {
        throw new LexerError('Unterminated quoted string', start);
      }
      continue;
    }

    const start = index;
    let value = '';
    while (index < input.length && !/\s/.test(input[index]) && !['(', ')', '=', '~', '"'].includes(input[index])) {
      value += input[index++];
    }

    if (!value) throw new LexerError(`Unexpected character "${char}"`, index);

    const keyword = KEYWORDS[value.toUpperCase()];
    tokens.push({
      type: keyword ?? 'identifier',
      value,
      position: start
    });
  }

  tokens.push({ type: 'eof', value: '', position: input.length });
  return tokens;
}
