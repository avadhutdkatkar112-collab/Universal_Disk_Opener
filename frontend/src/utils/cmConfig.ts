import { EditorView } from "@codemirror/view";
import { Extension } from "@codemirror/state";
import { StreamLanguage } from "@codemirror/language";

// Official CM6 language packages
import { javascript } from "@codemirror/lang-javascript";
import { java } from "@codemirror/lang-java";
import { json } from "@codemirror/lang-json";
import { cpp } from "@codemirror/lang-cpp";
import { php } from "@codemirror/lang-php";
import { python } from "@codemirror/lang-python";
import { go } from "@codemirror/lang-go";
import { css } from "@codemirror/lang-css";
import { sass } from "@codemirror/lang-sass";
import { less } from "@codemirror/lang-less";
import { html } from "@codemirror/lang-html";
import { sql } from "@codemirror/lang-sql";
import { rust } from "@codemirror/lang-rust";
import { xml } from "@codemirror/lang-xml";
import { markdown } from "@codemirror/lang-markdown";
import { yaml } from "@codemirror/lang-yaml";
import { vue } from "@codemirror/lang-vue";

// Legacy CM5 modes (for languages without official CM6 packages)
import { ruby } from "@codemirror/legacy-modes/mode/ruby";
import { lua } from "@codemirror/legacy-modes/mode/lua";
import { r } from "@codemirror/legacy-modes/mode/r";
import { perl } from "@codemirror/legacy-modes/mode/perl";
import { shell } from "@codemirror/legacy-modes/mode/shell";
import { erlang } from "@codemirror/legacy-modes/mode/erlang";
import { fortran } from "@codemirror/legacy-modes/mode/fortran";
import { pascal } from "@codemirror/legacy-modes/mode/pascal";
import { swift } from "@codemirror/legacy-modes/mode/swift";
import { clojure } from "@codemirror/legacy-modes/mode/clojure";
import { haskell } from "@codemirror/legacy-modes/mode/haskell";
import { cmake } from "@codemirror/legacy-modes/mode/cmake";
import { dockerFile } from "@codemirror/legacy-modes/mode/dockerfile";
import { diff } from "@codemirror/legacy-modes/mode/diff";
import { properties } from "@codemirror/legacy-modes/mode/properties";
import { nginx } from "@codemirror/legacy-modes/mode/nginx";
import { protobuf } from "@codemirror/legacy-modes/mode/protobuf";
import { coffeeScript } from "@codemirror/legacy-modes/mode/coffeescript";
import { powerShell } from "@codemirror/legacy-modes/mode/powershell";
import { vbScript } from "@codemirror/legacy-modes/mode/vbscript";
import { mathematica } from "@codemirror/legacy-modes/mode/mathematica";
import { octave } from "@codemirror/legacy-modes/mode/octave";
import { julia } from "@codemirror/legacy-modes/mode/julia";
import { groovy } from "@codemirror/legacy-modes/mode/groovy";
import { haxe } from "@codemirror/legacy-modes/mode/haxe";
import { d } from "@codemirror/legacy-modes/mode/d";
import { crystal } from "@codemirror/legacy-modes/mode/crystal";
import { elm } from "@codemirror/legacy-modes/mode/elm";
import { smalltalk } from "@codemirror/legacy-modes/mode/smalltalk";
import { sas } from "@codemirror/legacy-modes/mode/sas";
import { scheme } from "@codemirror/legacy-modes/mode/scheme";
import { verilog } from "@codemirror/legacy-modes/mode/verilog";
import { vhdl } from "@codemirror/legacy-modes/mode/vhdl";
import { webIDL } from "@codemirror/legacy-modes/mode/webidl";
import { ebnf } from "@codemirror/legacy-modes/mode/ebnf";
import { z80 } from "@codemirror/legacy-modes/mode/z80";
import { gas } from "@codemirror/legacy-modes/mode/gas";
import { tcl } from "@codemirror/legacy-modes/mode/tcl";

// Helper: wrap a legacy CM5 StreamParser into an Extension via StreamLanguage
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function lm(mode: any): Extension[] {
  return [StreamLanguage.define(mode)];
}

// ─── Theme ───────────────────────────────────────────────────────────────────

export const vhdExplorerTheme = EditorView.theme(
  {
    "&": {
      backgroundColor: "var(--bg-root)",
      color: "var(--text-primary)",
      fontSize: "12px",
      fontFamily: "var(--font-mono)",
      height: "100%",
    },
    ".cm-content": {
      padding: "8px 0",
    },
    ".cm-gutters": {
      backgroundColor: "var(--bg-surface)",
      color: "var(--text-disabled)",
      borderRight: "1px solid var(--border-subtle)",
    },
    ".cm-activeLineGutter": {
      backgroundColor: "var(--bg-elevated)",
      color: "var(--text-primary)",
    },
    ".cm-activeLine": {
      backgroundColor: "var(--bg-hover)",
    },
    ".cm-selectionBackground, ::selection": {
      backgroundColor: "var(--accent-muted) !important",
    },
    ".cm-cursor": {
      borderLeftColor: "var(--accent)",
    },
  },
  { dark: true }
);

// ─── Language Registry ───────────────────────────────────────────────────────
// Maps file extension → CodeMirror language extension
// Official CM6 packages preferred; legacy CM5 modes as fallback.
// Covers ~300+ file extensions across all major languages.

export function getLanguageExtension(ext: string): Extension[] {
  const e = ext.toLowerCase().replace(/^\./, "");

  // ── JavaScript / TypeScript ──────────────────────────────────────────────
  if (["js", "mjs", "cjs", "es", "es6", "es7", "jsx"].includes(e))
    return [javascript({ jsx: true })];
  if (["ts", "tsx", "mts", "cts"].includes(e))
    return [javascript({ jsx: true, typescript: true })];
  if (e === "vue") return [vue()];

  // ── Web / Markup ─────────────────────────────────────────────────────────
  if (["html", "htm", "xhtml", "shtml", "stm"].includes(e)) return [html()];
  if (["xml", "svg", "xsl", "xslt", "rss", "atom", "rdf", "wsdl", "plist",
       "xaml", "csproj", "vbproj", "fsproj", "vcxproj"].includes(e)) return [xml()];
  if (e === "css") return [css()];
  if (["scss", "sass"].includes(e)) return [sass()];
  if (e === "less") return [less()];
  if (["styl", "stylus"].includes(e)) return [less()];
  if (["md", "markdown", "mdown", "mkd", "mkdn", "mdwn", "mdtxt", "mdtext", "rst"].includes(e))
    return [markdown()];
  if (["liquid", "liquidjs"].includes(e)) return [html()];
  if (["pug", "jade"].includes(e)) return [html()];
  if (["ejs", "erb", "hbs", "handlebars", "mustache", "njk", "nunjucks", "twig", "jinja", "jinja2"].includes(e))
    return [html()];

  // ── JSON / Config ────────────────────────────────────────────────────────
  if (["json", "jsonl", "json5", "jsonc", "geojson", "topojson",
       "webmanifest", "bowerrc", "eslintrc", "prettierrc", "babelrc",
       "tsconfig", "jsconfig"].includes(e)) return [json()];
  if (["yaml", "yml", "yaml-tmlanguage", "clang-format"].includes(e)) return [yaml()];
  if (e === "toml") return [yaml()];
  if (["ini", "cfg", "conf", "config", "env", "properties", "desktop",
       "service", "socket", "timer", "mount", "target", "editorconfig",
       "sshconfig", "sshdconfig"].includes(e)) return lm(properties);
  if (["hcl", "tf", "tfvars", "nomad", "consul-template", "packer"].includes(e)) return [yaml()];
  if (["lock", "lockfile", "sum"].includes(e)) return [json()];

  // ── Python ───────────────────────────────────────────────────────────────
  if (["py", "pyw", "pyi", "py3", "py2", "gyp", "gypi", "wscript",
       "sconstruct", "python"].includes(e)) return [python()];

  // ── Java / JVM ───────────────────────────────────────────────────────────
  if (["java", "jav"].includes(e)) return [java()];
  if (["kt", "kts", "ktm"].includes(e)) return lm(groovy);
  if (["scala", "sc", "sbt"].includes(e)) return lm(groovy);
  if (["groovy", "gvy", "gy", "gsh"].includes(e)) return lm(groovy);
  if (["clj", "cljs", "cljc", "edn"].includes(e)) return lm(clojure);

  // ── Go ───────────────────────────────────────────────────────────────────
  if (["go", "gox"].includes(e)) return [go()];

  // ── Rust ─────────────────────────────────────────────────────────────────
  if (["rs", "rust", "rlib", "rmeta"].includes(e)) return [rust()];

  // ── C / C++ / Objective-C ────────────────────────────────────────────────
  if (["c", "h", "hc"].includes(e)) return [cpp()];
  if (["cpp", "cxx", "cc", "c++", "h++", "hpp", "hxx", "hh", "ii",
       "inl", "ipp", "tcc", "txx"].includes(e)) return [cpp()];

  // ── PHP ──────────────────────────────────────────────────────────────────
  if (["php", "php3", "php4", "php5", "php7", "php8", "phtml", "phps", "ctp"].includes(e))
    return [php()];

  // ── SQL / Database ──────────────────────────────────────────────────────
  if (["sql", "mysql", "pgsql", "postgres", "postgresql", "sqlite",
       "plsql", "tsql", "dml", "ddl", "dcl", "ctl"].includes(e)) return [sql()];

  // ── Shell / DevOps ──────────────────────────────────────────────────────
  if (["sh", "bash", "zsh", "ksh", "csh", "tcsh", "fish", "ash", "dash",
       "bashrc", "zshrc", "profile", "bash_profile", "bash_login",
       "bash_logout", "inputrc", "gitconfig", "gitignore", "gitattributes",
       "gitmodules", "dockerignore"].includes(e)) return lm(shell);
  if (["ps1", "psm1", "psd1", "ps1xml"].includes(e)) return lm(powerShell);
  if (["bat", "cmd", "btm", "nt"].includes(e)) return lm(shell);
  if (["dockerfile", "docker-compose", "docker-compose.yml"].includes(e))
    return lm(dockerFile);
  if (["makefile", "makefile.am", "makefile.in", "gnumake", "gmake", "mak"].includes(e))
    return lm(cmake);
  if (["cmake", "cmakein", "cmakecache"].includes(e)) return lm(cmake);
  if (["nginx", "nginxconf"].includes(e)) return lm(nginx);

  // ── Ruby ─────────────────────────────────────────────────────────────────
  if (["rb", "ruby", "rake", "rakefile", "gemspec", "gemfile", "rbx",
       "builder", "ru", "podspec", "thor", "vagrantfile"].includes(e))
    return lm(ruby);

  // ── Perl ─────────────────────────────────────────────────────────────────
  if (["pl", "pm", "perl", "t", "pod", "cpanfile"].includes(e)) return lm(perl);

  // ── Lua ──────────────────────────────────────────────────────────────────
  if (["lua", "lua5", "luac"].includes(e)) return lm(lua);

  // ── R / Statistics ──────────────────────────────────────────────────────
  if (["r", "rhistory", "rprofile", "rdata", "rda", "rds", "rmd", "rnw"].includes(e))
    return lm(r);

  // ── Haskell ──────────────────────────────────────────────────────────────
  if (["hs", "lhs", "hsc", "hsig", "cabal", "stack"].includes(e))
    return lm(haskell);

  // ── Erlang / Elixir ─────────────────────────────────────────────────────
  if (["erl", "hrl", "escript", "erlang"].includes(e)) return lm(erlang);
  if (["ex", "exs", "eex", "leex", "elixir"].includes(e)) return lm(ruby);

  // ── Swift ────────────────────────────────────────────────────────────────
  if (["swift", "swiftconfig"].includes(e)) return lm(swift);

  // ── Pascal / Delphi ─────────────────────────────────────────────────────
  if (["pas", "pp", "lpr", "dpr", "dpk", "dproj", "lpi", "lps",
       "dfm", "lfm", "lrs", "pascal", "delphi", "objectpascal"].includes(e))
    return lm(pascal);

  // ── Fortran ──────────────────────────────────────────────────────────────
  if (["f", "for", "f90", "f95", "f03", "f08", "f18", "fpp", "f77",
       "f15", "fortran"].includes(e)) return lm(fortran);

  // ── Assembly ─────────────────────────────────────────────────────────────
  if (["asm", "s", "asm68k", "nasm", "masm", "a51", "z80"].includes(e))
    return lm(z80);
  if (["gas", "s51", "a68k"].includes(e)) return lm(gas);

  // ── MATLAB / Octave ─────────────────────────────────────────────────────
  if (["matlab", "mat", "mex"].includes(e)) return lm(octave);
  if (["oct", "octave", "octaverc"].includes(e)) return lm(octave);

  // ── Julia ────────────────────────────────────────────────────────────────
  if (["jl", "julia", "jlproj"].includes(e)) return lm(julia);

  // ── Scheme / Lisp ───────────────────────────────────────────────────────
  if (["scm", "ss", "sls", "sps", "rkt", "rktl", "sch", "scheme"].includes(e))
    return lm(scheme);

  // ── CoffeeScript ─────────────────────────────────────────────────────────
  if (["coffee", "litcoffee"].includes(e)) return lm(coffeeScript);

  // ── Haxe ─────────────────────────────────────────────────────────────────
  if (["hx", "hxsl", "hxml", "haxe"].includes(e)) return lm(haxe);

  // ── D ────────────────────────────────────────────────────────────────────
  if (["d", "di", "dd", "dlang"].includes(e)) return lm(d);

  // ── Crystal ──────────────────────────────────────────────────────────────
  if (["cr", "crystal"].includes(e)) return lm(crystal);

  // ── Elm ──────────────────────────────────────────────────────────────────
  if (["elm", "elm-bu"].includes(e)) return lm(elm);

  // ── Smalltalk ────────────────────────────────────────────────────────────
  if (["st", "smalltalk", "squeak"].includes(e)) return lm(smalltalk);

  // ── SAS ──────────────────────────────────────────────────────────────────
  if (["sas", "sas7bdat"].includes(e)) return lm(sas);

  // ── Verilog / VHDL ──────────────────────────────────────────────────────
  if (["v", "vh", "sv", "svh", "verilog", "systemverilog"].includes(e))
    return lm(verilog);
  if (["vhd", "vhdl"].includes(e)) return lm(vhdl);

  // ── Protobuf ─────────────────────────────────────────────────────────────
  if (["proto", "protobuf"].includes(e)) return lm(protobuf);

  // ── Diff / Patch ─────────────────────────────────────────────────────────
  if (["diff", "patch", "rej", "gitpatch", "gitdiff"].includes(e))
    return lm(diff);

  // ── Mathematica / Wolfram ────────────────────────────────────────────────
  if (["nb", "wl", "wls"].includes(e)) return lm(mathematica);

  // ── WebIDL ───────────────────────────────────────────────────────────────
  if (["webidl", "widl"].includes(e)) return lm(webIDL);

  // ── EBNF / Grammar ──────────────────────────────────────────────────────
  if (["ebnf", "bnf", "abnf", "peg", "pegjs"].includes(e)) return lm(ebnf);

  // ── Groovy / Gradle ─────────────────────────────────────────────────────
  if (["gradle"].includes(e)) return lm(groovy);

  // ── LaTeX / TeX ─────────────────────────────────────────────────────────
  if (["tex", "latex", "ltx", "bib", "cls", "sty", "dtx", "ins",
       "lbx", "cbx", "bbx", "def"].includes(e)) return [markdown()];

  // ── Tcl ──────────────────────────────────────────────────────────────────
  if (["tcl", "wish", "itcl"].includes(e)) return lm(tcl);

  // ── VBScript ─────────────────────────────────────────────────────────────
  if (["vbs", "vbe", "wsc", "wsf"].includes(e)) return lm(vbScript);

  // ── Objective-C ──────────────────────────────────────────────────────────
  if (["m", "mm"].includes(e)) return lm(perl); // Fallback

  // ── Fallback: no highlighting ────────────────────────────────────────────
  return [];
}
