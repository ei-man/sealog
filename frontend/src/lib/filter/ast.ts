export type ConditionNode = {
  type: 'condition';
  field: string;
  operator: '=' | '~=';
  value: string;
  regex?: RegExp;
};

export type AndNode = {
  type: 'and';
  left: AstNode;
  right: AstNode;
};

export type OrNode = {
  type: 'or';
  left: AstNode;
  right: AstNode;
};

export type NotNode = {
  type: 'not';
  expr: AstNode;
};

export type AstNode = ConditionNode | AndNode | OrNode | NotNode;
