/**
 * Generic StreamParser fallback for unknown file extensions.
 * Provides basic syntax highlighting using regex rules — strings, comments,
 * numbers, keywords, and structural brackets. Used when no TextMate grammar
 * is available (neither bundled nor fetched from CDN).
 */
import { StreamLanguage, StreamParser } from "@codemirror/language";

// Token types used by CodeMirror's built-in highlight style
const TOKENS: Record<string, string> = {
  string: "string",
  comment: "comment",
  number: "number",
  keyword: "keyword",
  type: "type",
  operator: "operator",
  variable: "variableName",
  property: "propertyName",
  atom: "atom",
  bracket: "bracket",
  meta: "meta",
};

/**
 * Creates a StreamParser for generic highlighting based on common patterns.
 * This covers ~80% of text-based config/source files well enough for browsing.
 */
const genericParser: StreamParser<{ inBlockComment: boolean }> = {
  startState: () => ({ inBlockComment: false }),

  token(stream, state) {
    // Block comment tracking (/* ... */)
    if (state.inBlockComment) {
      if (stream.match("*/")) {
        state.inBlockComment = false;
      } else {
        stream.next();
      }
      return TOKENS.comment;
    }

    // Skip whitespace
    if (stream.eatSpace()) return null;

    // Line comment: // # ; -- % """
    if (stream.match("//") || stream.match("#") || stream.match(";") || stream.match("--") || stream.match("%")) {
      stream.skipToEnd();
      return TOKENS.comment;
    }

    // Block comment: /* ... */
    if (stream.match("/*")) {
      state.inBlockComment = true;
      return TOKENS.comment;
    }

    // Triple-quote strings: """ or '''
    if (stream.match('"""') || stream.match("'''")) {
      const quote = stream.current();
      while (!stream.eol()) {
        if (stream.match(quote)) break;
        stream.next();
      }
      return TOKENS.string;
    }

    // Strings: "..." or '...'
    if (stream.match('"') || stream.match("'")) {
      const quote = stream.current();
      while (!stream.eol()) {
        const ch = stream.next();
        if (ch === "\\") {
          stream.next(); // skip escaped char
        } else if (ch === quote[0]) {
          break;
        }
      }
      return TOKENS.string;
    }

    // Backtick template strings
    if (stream.match("`")) {
      while (!stream.eol()) {
        const ch = stream.next();
        if (ch === "\\") {
          stream.next();
        } else if (ch === "`") {
          break;
        }
      }
      return TOKENS.string;
    }

    // Numbers: integers, floats, hex, binary
    if (stream.match(/^-?(0[xX][0-9a-fA-F]+|0[bB][01]+|0[oO][0-7]+|\d+\.?\d*([eE][+-]?\d+)?)/)) {
      return TOKENS.number;
    }

    // Keywords (common across many languages)
    if (stream.match(/^(if|else|elif|elseif|then|fi|for|while|do|done|switch|case|esac|function|return|exit|import|from|as|class|struct|enum|type|interface|extends|implements|public|private|protected|static|final|const|let|var|fn|def|func|package|module|using|namespace|try|catch|finally|throw|throws|new|delete|typeof|instanceof|in|of|with|yield|async|await|lambda|match|when|where|select|insert|update|delete|create|drop|alter|table|index|view|grant|revoke|begin|commit|rollback|select|from|where|join|left|right|inner|outer|on|group|order|by|having|limit|offset|union|all|distinct|as|into|values|set|and|or|not|null|true|false|nil|none|undefined|NaN|Infinity|self|this|super|parent|root|main|entry|start|init|setup|teardown|before|after|describe|it|test|expect|assert|should|must|required|optional|abstract|virtual|override|inline|extern|static|dynamic|lazy|readonly|writeonly|volatile|atomic|sync|async|defer|panic|recover|assert|require|ensure|precondition|postcondition|invariant|variant|specialize|generic|impl|trait|derive|macro|template|typename|concept|requires|co|await|module|exports|define|undef|ifdef|ifndef|pragma|error|warning|info|debug|trace|log|print|println|echo|printf|fprintf|sprintf|scan|read|write|open|close|seek|tell|flush|sync|truncate|chown|chmod|rename|mkdir|rmdir|unlink|stat|lstat|fstat|link|symlink|readlink|realpath|dirname|basename|path|abs|ceil|floor|round|sqrt|pow|log|exp|sin|cos|tan|asin|acos|atan|atan2|min|max|abs|sign|length|size|count|sum|avg|mean|median|mode|range|sort|reverse|unique|flatten|compact|map|filter|reduce|fold|each|every|some|any|none|contains|includes|find|index|first|last|head|tail|take|drop|zip|concat|append|prepend|insert|remove|delete|clear|empty|null|undefined|true|false|yes|no|on|off|enable|disable|start|stop|pause|resume|cancel|retry|abort|skip|pass|fail|ok|error|warn|info|debug|fatal|panic|critical|notice|alert|emergency)$/)) {
      return TOKENS.keyword;
    }

    // Types / capitalized identifiers (PascalCase = likely a type)
    if (stream.match(/^[A-Z][a-zA-Z0-9_]*$/)) {
      return TOKENS.type;
    }

    // Operators
    if (stream.match(/^(==|!=|<=|>=|<>|&&|\|\||<<|>>|>>>|\*\*|\+=|-=|\*=|\/=|%=|&=|\|=|\^=|=>|->|<-|::|\.\.\.|<<=|>>=)/)) {
      return TOKENS.operator;
    }

    // Brackets
    if (stream.match(/^([{}()\[\]])/)) {
      return TOKENS.bracket;
    }

    // Identifiers (default)
    if (stream.match(/^[a-zA-Z_$][a-zA-Z0-9_$]*/)) {
      return TOKENS.variable;
    }

    // Everything else
    stream.next();
    return null;
  },
};

/**
 * Creates a StreamLanguage from the generic parser.
 * This is the Tier 3 fallback when no grammar is available.
 */
export const genericHighlight = StreamLanguage.define(genericParser);
