(window.webpackJsonp=window.webpackJsonp||[]).push([[7,47],{151:function(e,t,a){"use strict";a.r(t);a(12);var n=a(0),l=a.n(n),r=a(155),c=a(152),i=a(47),o=a(160),s=a(1),m=a(153),u=a.n(m),d=a(162),b=a(163),h=a(154),g=a(186),E=a(108),p=a.n(E);function f({item:e,onItemClick:t,collapsible:a,...r}){const{items:c,href:i,label:o,type:m}=e,[d,b]=Object(n.useState)(e.collapsed),[E,p]=Object(n.useState)(null);e.collapsed!==E&&(p(e.collapsed),b(e.collapsed));const k=Object(n.useCallback)(e=>{e.preventDefault(),e.target.blur(),b(e=>!e)});switch(m){case"category":return c.length>0&&l.a.createElement("li",{className:u()("menu__list-item",{"menu__list-item--collapsed":d}),key:o},l.a.createElement("a",Object(s.a)({className:u()("menu__link",{"menu__link--sublist":a,"menu__link--active":a&&!e.collapsed}),href:"#!",onClick:a?k:void 0},r),o),l.a.createElement("ul",{className:"menu__list"},c.map(e=>l.a.createElement(f,{tabIndex:d?"-1":"0",key:e.label,item:e,onItemClick:t,collapsible:a}))));case"link":default:return l.a.createElement("li",{className:"menu__list-item",key:o},l.a.createElement(h.a,Object(s.a)({className:"menu__link",to:i},Object(g.a)(i)?{isNavLink:!0,activeClassName:"menu__link--active",exact:!0,onClick:t}:{target:"_blank",rel:"noreferrer noopener"},r),o))}}var k=function(e){const[t,a]=Object(n.useState)(!1),{siteConfig:{themeConfig:{navbar:{title:r,hideOnScroll:i=!1}={}}}={},isClient:o}=Object(c.a)(),{logoLink:m,logoLinkProps:g,logoImageUrl:E,logoAlt:k}=Object(b.a)(),{docsSidebars:v,path:_,sidebar:N,sidebarCollapsible:y}=e;if(Object(d.a)(t),!N)return null;const w=v[N];if(!w)throw new Error(`Cannot find the sidebar "${N}" in the sidebar config!`);return y&&w.forEach(e=>function e(t,a){const{items:n,href:l,type:r}=t;switch(r){case"category":{const l=n.map(t=>e(t,a)).filter(e=>e).length>0;return t.collapsed=!l,l}case"link":default:return l===a}}(e,_)),l.a.createElement("div",{className:p.a.sidebar},i&&l.a.createElement(h.a,Object(s.a)({tabIndex:"-1",className:p.a.sidebarLogo,to:m},g),null!=E&&l.a.createElement("img",{key:o,src:E,alt:k}),null!=r&&l.a.createElement("strong",null,r)),l.a.createElement("div",{className:u()("menu","menu--responsive",p.a.menu,{"menu--show":t})},l.a.createElement("button",{"aria-label":t?"Close Menu":"Open Menu","aria-haspopup":"true",className:"button button--secondary button--sm menu__button",type:"button",onClick:()=>{a(!t)}},t?l.a.createElement("span",{className:u()(p.a.sidebarMenuIcon,p.a.sidebarMenuCloseIcon)},"\xd7"):l.a.createElement("svg",{"aria-label":"Menu",className:p.a.sidebarMenuIcon,xmlns:"http://www.w3.org/2000/svg",height:24,width:24,viewBox:"0 0 32 32",role:"img",focusable:"false"},l.a.createElement("title",null,"Menu"),l.a.createElement("path",{stroke:"currentColor",strokeLinecap:"round",strokeMiterlimit:"10",strokeWidth:"2",d:"M4 7h22M4 15h22M4 23h22"}))),l.a.createElement("ul",{className:"menu__list"},w.map(e=>l.a.createElement(f,{key:e.label,item:e,onItemClick:e=>{e.target.blur(),a(!1)},collapsible:y})))))},v=a(212),_=a(209),N=a(180),y=a(110),w=a.n(y);t.default=function(e){const{route:t,docsMetadata:a,location:n}=e,s=t.routes.find(e=>Object(N.b)(n.pathname,e))||{},{permalinkToSidebar:m,docsSidebars:u,version:d}=a,b=m[s.path],{siteConfig:{themeConfig:h={}}={},isClient:g}=Object(c.a)(),{sidebarCollapsible:E=!0}=h;return 0===Object.keys(s).length?l.a.createElement(_.default,e):l.a.createElement(o.a,{version:d,key:g},l.a.createElement("div",{className:w.a.docPage},b&&l.a.createElement("div",{className:w.a.docSidebarContainer},l.a.createElement(k,{docsSidebars:u,path:s.path,sidebar:b,sidebarCollapsible:E})),l.a.createElement("main",{className:w.a.docMainContainer},l.a.createElement(r.a,{components:v.a},Object(i.a)(t.routes)))))}},159:function(e,t,a){"use strict";a.d(t,"a",(function(){return l})),a.d(t,"b",(function(){return r}));var n=a(152);function l(){const e=Object(n.a)(),{siteConfig:t={}}=e,{metadata:{latest_highlight:a}}=t.customFields,l=Date.parse(a.date),r=new Date,c=Math.abs(r-l),i=Math.ceil(c/864e5);let o=null;return"undefined"!=typeof window&&(o=new Date(parseInt(window.localStorage.getItem("highlightsViewedAt")||"0"))),i<30&&(!o||o<l)?a:null}function r(){"undefined"!=typeof window&&window.localStorage.setItem("highlightsViewedAt",(new Date).getTime())}},160:function(e,t,a){"use strict";var n=a(0),l=a.n(n),r=a(165),c=a(157),i=a(1),o=(a(166),a(154)),s=a(167),m=a(158),u=a.n(m),d=a(168),b=a.n(d),h=a(152),g=a(153),E=a.n(g),p=a(93),f=a.n(p);const k=()=>l.a.createElement("span",{className:E()(f.a.toggle,f.a.moon)}),v=()=>l.a.createElement("span",{className:E()(f.a.toggle,f.a.sun)});var _=function(e){const{isClient:t}=Object(h.a)();return l.a.createElement(b.a,Object(i.a)({disabled:!t,icons:{checked:l.a.createElement(k,null),unchecked:l.a.createElement(v,null)}},e))},N=a(159),y=a(156),w=a(172),O=a(162),j=a(163),C=a(161),S=a(94),T=a.n(S);function x({href:e,hideIcon:t,label:a,onClick:n,position:r,right:c,to:s}){let m=function(e,t){let a={label:e};switch(e.toLowerCase()){case"chat":return a.hideText=1==t,a.icon="message-circle",a;case"community":return a.hideText=1==t,a.icon="users",a;case"github":return a.badge="3k",a.hideText=!1,a.icon="github",a;case"highlights":return Object(N.a)()&&(a.badge="new",a.badgeStyle="primary"),a.hideText=1==t,a.icon="gift",a;default:return a}}(a,c)||{};const u=Object(y.a)(s);return l.a.createElement(o.a,Object(i.a)({className:E()("navbar__item navbar__link",m.className,{navbar__item__icon_only:m.hideText}),title:m.hideText?a:null,onClick:n},e?{target:"_blank",rel:"noopener noreferrer",href:e}:{activeClassName:"navbar__link--active",to:u}),!t&&m.icon&&l.a.createElement(l.a.Fragment,null,l.a.createElement("i",{className:"feather icon-"+m.icon})," ",m.iconLabel),!m.hideText&&m.label,m.badge&&l.a.createElement("span",{className:E()("badge","badge--"+(m.badgeStyle||"secondary"))},m.badge))}var M=function(){const{siteConfig:{themeConfig:{navbar:{title:e,links:t=[],hideOnScroll:a=!1}={},disableDarkMode:r=!1}},isClient:c}=Object(h.a)(),[m,d]=Object(n.useState)(!1),[b,g]=Object(n.useState)(!1),{isDarkTheme:p,setLightTheme:f,setDarkTheme:k}=Object(C.a)(),{navbarRef:v,isNavbarVisible:N}=Object(w.a)(a),{logoLink:y,logoLinkProps:S,logoImageUrl:M,logoAlt:I}=Object(j.a)();Object(O.a)(m);const L=Object(n.useCallback)(()=>{d(!0)},[d]),P=Object(n.useCallback)(()=>{d(!1)},[d]),D=Object(n.useCallback)(e=>e.target.checked?k():f(),[f,k]);return l.a.createElement("nav",{ref:v,className:E()("navbar","navbar--light","navbar--fixed-top",{"navbar-sidebar--show":m,[T.a.navbarHideable]:a,[T.a.navbarHidden]:!N})},l.a.createElement("div",{className:"navbar__inner"},l.a.createElement("div",{className:"navbar__items"},l.a.createElement("div",{"aria-label":"Navigation bar toggle",className:"navbar__toggle",role:"button",tabIndex:0,onClick:L,onKeyDown:L},l.a.createElement("svg",{xmlns:"http://www.w3.org/2000/svg",width:"30",height:"30",viewBox:"0 0 30 30",role:"img",focusable:"false"},l.a.createElement("title",null,"Menu"),l.a.createElement("path",{stroke:"currentColor",strokeLinecap:"round",strokeMiterlimit:"10",strokeWidth:"2",d:"M4 7h22M4 15h22M4 23h22"}))),l.a.createElement(o.a,Object(i.a)({className:"navbar__brand",to:y},S),null!=M&&l.a.createElement(u.a,{className:"navbar__logo",src:M,alt:I}),null!=e&&l.a.createElement("strong",{className:b?T.a.hideLogoText:""},e)),t.filter(e=>"right"!==e.position).map((e,t)=>l.a.createElement(x,Object(i.a)({},e,{left:!0,key:t})))),l.a.createElement("div",{className:"navbar__items navbar__items--right"},t.filter(e=>"right"===e.position).map((e,t)=>l.a.createElement(x,Object(i.a)({},e,{right:!0,key:t}))),!r&&l.a.createElement(_,{className:T.a.displayOnlyInLargeViewport,"aria-label":"Dark mode toggle",checked:p,onChange:D}),l.a.createElement(s.a,{handleSearchBarToggle:g,isSearchBarExpanded:b}))),l.a.createElement("div",{role:"presentation",className:"navbar-sidebar__backdrop",onClick:P}),l.a.createElement("div",{className:"navbar-sidebar"},l.a.createElement("div",{className:"navbar-sidebar__brand"},l.a.createElement(o.a,Object(i.a)({className:"navbar__brand",onClick:P,to:y},S),null!=M&&l.a.createElement(u.a,{className:"navbar__logo",src:M,alt:I}),null!=e&&l.a.createElement("strong",null,e)),!r&&m&&l.a.createElement(_,{"aria-label":"Dark mode toggle in sidebar",checked:p,onChange:D})),l.a.createElement("div",{className:"navbar-sidebar__items"},l.a.createElement("div",{className:"menu"},l.a.createElement("ul",{className:"menu__list"},t.map((e,t)=>l.a.createElement("li",{className:"menu__list-item",key:t},l.a.createElement(x,Object(i.a)({className:"menu__link"},e,{hideIcon:!0,onClick:P})))))))))};a(95);var I=function({block:e,buttonClass:t,center:a,description:n,size:r,width:c}){return l.a.createElement("div",{className:E()("mailing-list",{"mailing-list--block":e,"mailing-list--center":a,["mailing-list--"+r]:r})},!1!==n&&l.a.createElement("div",{className:"mailing-list--description"},"The easiest way to stay up-to-date. One email on the 1st of every month. No spam, ever."),l.a.createElement("form",{action:"https://app.getvero.com/forms/a748ded7ce0da69e6042fa1e21042506",method:"post",className:"mailing-list--form"},l.a.createElement("input",{className:E()("input","input--"+r),name:"email",placeholder:"you@email.com",type:"email",style:{width:c}}),l.a.createElement("button",{className:E()("button","button--"+(t||"primary"),"button--"+r),type:"submit"},"Subscribe")))},L=a(96),P=a.n(L);function D({to:e,href:t,label:a,...n}){const r=Object(y.a)(e);return l.a.createElement(o.a,Object(i.a)({className:"footer__link-item"},t?{target:"_blank",rel:"noopener noreferrer",href:t}:{to:r},n),a)}const B=({url:e,alt:t})=>l.a.createElement(u.a,{className:"footer__logo",alt:t,src:e});var A=function(){const e=Object(h.a)(),{siteConfig:t={}}=e,{themeConfig:a={}}=t,{footer:n}=a,{copyright:r,links:c=[],logo:i={}}=n||{},o=Object(y.a)(i.src);return n?l.a.createElement("footer",{className:E()("footer",{"footer--dark":"dark"===n.style})},l.a.createElement("div",{className:"container"},c&&c.length>0&&l.a.createElement("div",{className:"row footer__links"},l.a.createElement("div",{className:"col col--5 footer__col"},l.a.createElement("div",{className:"margin-bottom--md"},l.a.createElement(u.a,{className:"navbar__logo",src:"/img/logo-light.svg",alt:"Gnet",width:"150",height:"auto"})),l.a.createElement("div",{className:"margin-bottom--md"},l.a.createElement(I,{description:!1,width:"150px"})),l.a.createElement("div",null,l.a.createElement("a",{href:"https://twitter.com/_andy_pan",target:"_blank"},l.a.createElement("i",{className:"feather icon-twitter",alt:"Gnet's Twitter"})),"\xa0\xa0\xa0\xa0",l.a.createElement("a",{href:"https://github.com/panjf2000/gnet",target:"_blank"},l.a.createElement("i",{className:"feather icon-github",alt:"Gnet's Github Repo"})),"\xa0\xa0\xa0\xa0",l.a.createElement("a",{href:"https://taohuawu.club/rss.xml",target:"_blank"},l.a.createElement("i",{className:"feather icon-rss",alt:"Gnet's RSS feed"})))),c.map((e,t)=>l.a.createElement("div",{key:t,className:"col footer__col"},null!=e.title?l.a.createElement("h4",{className:"footer__title"},e.title):null,null!=e.items&&Array.isArray(e.items)&&e.items.length>0?l.a.createElement("ul",{className:"footer__items"},e.items.map((e,t)=>e.html?l.a.createElement("li",{key:t,className:"footer__item",dangerouslySetInnerHTML:{__html:e.html}}):l.a.createElement("li",{key:e.href||e.to,className:"footer__item"},l.a.createElement(D,e)))):null))),(i||r)&&l.a.createElement("div",{className:"text--center"},i&&i.src&&l.a.createElement("div",{className:"margin-bottom--sm"},i.href?l.a.createElement("a",{href:i.href,target:"_blank",rel:"noopener noreferrer",className:P.a.footerLogoLink},l.a.createElement(B,{alt:i.alt,url:o})):l.a.createElement(B,{alt:i.alt,url:o})),r,l.a.createElement("br",null),l.a.createElement("small",null,l.a.createElement("a",{href:"https://github.com/panjf2000/gnet/security/policy"},"Security Policy"),"\xa0\u2022\xa0",l.a.createElement("a",{href:"https://github.com/panjf2000/gnet/blob/master/PRIVACY.md"},"Privacy Policy")),l.a.createElement("br",null),l.a.createElement("small",null,"Acknowledgement for Timber, Inc.")))):null},R=a(173),F=a(174),W=a(2);a(97);t.a=function(e){const{siteConfig:t={}}=Object(h.a)(),{favicon:a,tagline:n,title:i,themeConfig:{image:o},url:s}=t,{children:m,title:u,noFooter:d,description:b,image:g,keywords:E,permalink:p,version:f}=e,k=u?`${u} | ${i}`:i,v=g||o,_=s+Object(y.a)(v),N=Object(y.a)(a),w=Object(W.h)();let O=w?"https://gnet.host"+(w.pathname.endsWith("/")?w.pathname:w.pathname+"/"):null;return l.a.createElement(F.a,null,l.a.createElement(R.a,null,l.a.createElement(c.a,null,l.a.createElement("html",{lang:"en"}),l.a.createElement("meta",{httpEquiv:"x-ua-compatible",content:"ie=edge"}),k&&l.a.createElement("title",null,k),k&&l.a.createElement("meta",{property:"og:title",content:k}),a&&l.a.createElement("link",{rel:"shortcut icon",href:N}),b&&l.a.createElement("meta",{name:"description",content:b}),b&&l.a.createElement("meta",{property:"og:description",content:b}),f&&l.a.createElement("meta",{name:"docsearch:version",content:f}),E&&E.length&&l.a.createElement("meta",{name:"keywords",content:E.join(",")}),v&&l.a.createElement("meta",{property:"og:image",content:_}),v&&l.a.createElement("meta",{property:"twitter:image",content:_}),v&&l.a.createElement("meta",{name:"twitter:image:alt",content:"Image for "+k}),v&&l.a.createElement("meta",{name:"twitter:site",content:"@vectordotdev"}),v&&l.a.createElement("meta",{name:"twitter:creator",content:"@vectordotdev"}),O&&l.a.createElement("meta",{property:"og:url",content:O}),l.a.createElement("meta",{name:"twitter:card",content:"summary"}),O&&l.a.createElement("link",{rel:"canonical",href:O})),l.a.createElement(r.a,null),l.a.createElement(M,null),l.a.createElement("div",{className:"main-wrapper"},m),!d&&l.a.createElement(A,null)))}},187:function(e,t,a){"use strict";(function(e){var n=a(1),l=a(0),r=a.n(l),c=a(188),i=a.n(c),o=a(203),s=a(35),m=a(153),u=a.n(m),d=a(196),b=a(189),h=a.n(b),g=a(152),E=a(161),p=a(98),f=a.n(p);(void 0!==e?e:window).Prism=s.a,a(190),a(191),a(192),a(193),a(194),a(195);const k=/{([\d,-]+)}/,v=/title=".*"/;t.a=({children:e,className:t,metastring:a})=>{const{siteConfig:{themeConfig:{prism:c={}}}}=Object(g.a)(),[s,m]=Object(l.useState)(!1),[b,p]=Object(l.useState)(!1);Object(l.useEffect)(()=>{p(!0)},[]);const _=Object(l.useRef)(null),N=Object(l.useRef)(null);let y=[],w="";const{isDarkTheme:O}=Object(E.a)(),j=c.theme||d.a,C=c.darkTheme||j,S=O?C:j;if(a&&k.test(a)){const e=a.match(k)[1];y=h.a.parse(e).filter(e=>e>0)}a&&v.test(a)&&(w=a.match(v)[0].split("title=")[1].replace(/"+/g,"")),Object(l.useEffect)(()=>{let e;return N.current&&(e=new i.a(N.current,{target:()=>_.current})),()=>{e&&e.destroy()}},[N.current,_.current]);let T=t&&t.replace(/language-/,"");!T&&c.defaultLanguage&&(T=c.defaultLanguage);const x=()=>{window.getSelection().empty(),m(!0),setTimeout(()=>m(!1),2e3)};return r.a.createElement(o.a,Object(n.a)({},o.b,{key:b,theme:S,code:e.trim(),language:T}),({className:e,style:t,tokens:a,getLineProps:l,getTokenProps:c})=>r.a.createElement(r.a.Fragment,null,w&&r.a.createElement("div",{style:t,className:f.a.codeBlockTitle},w),r.a.createElement("div",{className:f.a.codeBlockContent},r.a.createElement("button",{ref:N,type:"button","aria-label":"Copy code to clipboard",className:u()(f.a.copyButton,{[f.a.copyButtonWithTitle]:w}),onClick:x},s?"Copied":"Copy"),r.a.createElement("pre",{className:u()(e,f.a.codeBlock,{[f.a.codeBlockWithTitle]:w})},r.a.createElement("div",{ref:_,className:f.a.codeBlockLines,style:t},a.map((e,t)=>{1===e.length&&""===e[0].content&&(e[0].content="\n");const a=l({line:e,key:t});return y.includes(t+1)&&(a.className=a.className+" docusaurus-highlight-code-line"),r.a.createElement("div",Object(n.a)({key:t},a),e.map((e,t)=>r.a.createElement("span",Object(n.a)({key:t},c({token:e,key:t})))))}))))))}}).call(this,a(52))},209:function(e,t,a){"use strict";a.r(t);var n=a(0),l=a.n(n),r=a(160);t.default=function(){return l.a.createElement(r.a,{title:"Page Not Found"},l.a.createElement("div",{className:"container margin-vert--xl"},l.a.createElement("div",{className:"row"},l.a.createElement("div",{className:"col col--6 col--offset-3"},l.a.createElement("h1",{className:"hero__title"},"Page Not Found"),l.a.createElement("p",null,"We could not find what you were looking for."),l.a.createElement("p",null,"Please contact the owner of the site that linked you to the original URL and let them know their link is broken.")))))}}}]);