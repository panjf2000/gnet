(window.webpackJsonp=window.webpackJsonp||[]).push([[19],{122:function(e,t,a){"use strict";a.r(t);var n=a(0),r=a.n(n),u=a(208),i=a(160),l=a(154);t.default=function(e){const{metadata:t,items:a}=e,{allTagsPath:n,name:o,count:s}=t;return r.a.createElement(i.a,{title:`Highlights tagged "${o}"`,description:`Highlight | Tagged "${o}"`},r.a.createElement("header",{className:"hero hero--clean"},r.a.createElement("div",{className:"container"},r.a.createElement("h1",null,s," ",function(e,t){return e>1?t+"s":t}(s,"highlight"),' tagged with "',o,'"'),r.a.createElement("div",{className:"hero--subtitle"},r.a.createElement(l.a,{href:n},"View All Tags")))),r.a.createElement("main",{className:"container container--xs"},r.a.createElement(u.a,{items:a})))}},159:function(e,t,a){"use strict";a.d(t,"a",(function(){return r})),a.d(t,"b",(function(){return u}));var n=a(152);function r(){const e=Object(n.a)(),{siteConfig:t={}}=e,{metadata:{latest_highlight:a}}=t.customFields,r=Date.parse(a.date),u=new Date,i=Math.abs(u-r),l=Math.ceil(i/864e5);let o=null;return"undefined"!=typeof window&&(o=new Date(parseInt(window.localStorage.getItem("highlightsViewedAt")||"0"))),l<30&&(!o||o<r)?a:null}function u(){"undefined"!=typeof window&&window.localStorage.setItem("highlightsViewedAt",(new Date).getTime())}},160:function(e,t,a){"use strict";var n=a(0),r=a.n(n),u=a(165),i=a(157),l=a(1),o=(a(166),a(154)),s=a(167),c=a(158),D=a.n(c),m=a(168),d=a.n(m),h=a(152),g=a(153),p=a.n(g),E=a(93),f=a.n(E);const b=()=>r.a.createElement("span",{className:p()(f.a.toggle,f.a.moon)}),y=()=>r.a.createElement("span",{className:p()(f.a.toggle,f.a.sun)});var F=function(e){const{isClient:t}=Object(h.a)();return r.a.createElement(d.a,Object(l.a)({disabled:!t,icons:{checked:r.a.createElement(b,null),unchecked:r.a.createElement(y,null)}},e))},v=a(159),C=a(156),w=a(172),$=a(162),k=a(163),N=a(161),_=a(94),A=a.n(_);function O({href:e,hideIcon:t,label:a,onClick:n,position:u,right:i,to:s}){let c=function(e,t){let a={label:e};switch(e.toLowerCase()){case"chat":return a.hideText=1==t,a.icon="message-circle",a;case"community":return a.hideText=1==t,a.icon="users",a;case"github":return a.badge="3k",a.hideText=!1,a.icon="github",a;case"highlights":return Object(v.a)()&&(a.badge="new",a.badgeStyle="primary"),a.hideText=1==t,a.icon="gift",a;default:return a}}(a,i)||{};const D=Object(C.a)(s);return r.a.createElement(o.a,Object(l.a)({className:p()("navbar__item navbar__link",c.className,{navbar__item__icon_only:c.hideText}),title:c.hideText?a:null,onClick:n},e?{target:"_blank",rel:"noopener noreferrer",href:e}:{activeClassName:"navbar__link--active",to:D}),!t&&c.icon&&r.a.createElement(r.a.Fragment,null,r.a.createElement("i",{className:"feather icon-"+c.icon})," ",c.iconLabel),!c.hideText&&c.label,c.badge&&r.a.createElement("span",{className:p()("badge","badge--"+(c.badgeStyle||"secondary"))},c.badge))}var T=function(){const{siteConfig:{themeConfig:{navbar:{title:e,links:t=[],hideOnScroll:a=!1}={},disableDarkMode:u=!1}},isClient:i}=Object(h.a)(),[c,m]=Object(n.useState)(!1),[d,g]=Object(n.useState)(!1),{isDarkTheme:E,setLightTheme:f,setDarkTheme:b}=Object(N.a)(),{navbarRef:y,isNavbarVisible:v}=Object(w.a)(a),{logoLink:C,logoLinkProps:_,logoImageUrl:T,logoAlt:M}=Object(k.a)();Object($.a)(c);const j=Object(n.useCallback)(()=>{m(!0)},[m]),x=Object(n.useCallback)(()=>{m(!1)},[m]),S=Object(n.useCallback)(e=>e.target.checked?b():f(),[f,b]);return r.a.createElement("nav",{ref:y,className:p()("navbar","navbar--light","navbar--fixed-top",{"navbar-sidebar--show":c,[A.a.navbarHideable]:a,[A.a.navbarHidden]:!v})},r.a.createElement("div",{className:"navbar__inner"},r.a.createElement("div",{className:"navbar__items"},r.a.createElement("div",{"aria-label":"Navigation bar toggle",className:"navbar__toggle",role:"button",tabIndex:0,onClick:j,onKeyDown:j},r.a.createElement("svg",{xmlns:"http://www.w3.org/2000/svg",width:"30",height:"30",viewBox:"0 0 30 30",role:"img",focusable:"false"},r.a.createElement("title",null,"Menu"),r.a.createElement("path",{stroke:"currentColor",strokeLinecap:"round",strokeMiterlimit:"10",strokeWidth:"2",d:"M4 7h22M4 15h22M4 23h22"}))),r.a.createElement(o.a,Object(l.a)({className:"navbar__brand",to:C},_),null!=T&&r.a.createElement(D.a,{className:"navbar__logo",src:T,alt:M}),null!=e&&r.a.createElement("strong",{className:d?A.a.hideLogoText:""},e)),t.filter(e=>"right"!==e.position).map((e,t)=>r.a.createElement(O,Object(l.a)({},e,{left:!0,key:t})))),r.a.createElement("div",{className:"navbar__items navbar__items--right"},t.filter(e=>"right"===e.position).map((e,t)=>r.a.createElement(O,Object(l.a)({},e,{right:!0,key:t}))),!u&&r.a.createElement(F,{className:A.a.displayOnlyInLargeViewport,"aria-label":"Dark mode toggle",checked:E,onChange:S}),r.a.createElement(s.a,{handleSearchBarToggle:g,isSearchBarExpanded:d}))),r.a.createElement("div",{role:"presentation",className:"navbar-sidebar__backdrop",onClick:x}),r.a.createElement("div",{className:"navbar-sidebar"},r.a.createElement("div",{className:"navbar-sidebar__brand"},r.a.createElement(o.a,Object(l.a)({className:"navbar__brand",onClick:x,to:C},_),null!=T&&r.a.createElement(D.a,{className:"navbar__logo",src:T,alt:M}),null!=e&&r.a.createElement("strong",null,e)),!u&&c&&r.a.createElement(F,{"aria-label":"Dark mode toggle in sidebar",checked:E,onChange:S})),r.a.createElement("div",{className:"navbar-sidebar__items"},r.a.createElement("div",{className:"menu"},r.a.createElement("ul",{className:"menu__list"},t.map((e,t)=>r.a.createElement("li",{className:"menu__list-item",key:t},r.a.createElement(O,Object(l.a)({className:"menu__link"},e,{hideIcon:!0,onClick:x})))))))))};a(95);var M=function({block:e,buttonClass:t,center:a,description:n,size:u,width:i}){return r.a.createElement("div",{className:p()("mailing-list",{"mailing-list--block":e,"mailing-list--center":a,["mailing-list--"+u]:u})},!1!==n&&r.a.createElement("div",{className:"mailing-list--description"},"The easiest way to stay up-to-date. One email on the 1st of every month. No spam, ever."),r.a.createElement("form",{action:"https://app.getvero.com/forms/a748ded7ce0da69e6042fa1e21042506",method:"post",className:"mailing-list--form"},r.a.createElement("input",{className:p()("input","input--"+u),name:"email",placeholder:"you@email.com",type:"email",style:{width:i}}),r.a.createElement("button",{className:p()("button","button--"+(t||"primary"),"button--"+u),type:"submit"},"Subscribe")))},j=a(96),x=a.n(j);function S({to:e,href:t,label:a,...n}){const u=Object(C.a)(e);return r.a.createElement(o.a,Object(l.a)({className:"footer__link-item"},t?{target:"_blank",rel:"noopener noreferrer",href:t}:{to:u},n),a)}const B=({url:e,alt:t})=>r.a.createElement(D.a,{className:"footer__logo",alt:t,src:e});var P=function(){const e=Object(h.a)(),{siteConfig:t={}}=e,{themeConfig:a={}}=t,{footer:n}=a,{copyright:u,links:i=[],logo:l={}}=n||{},o=Object(C.a)(l.src);return n?r.a.createElement("footer",{className:p()("footer",{"footer--dark":"dark"===n.style})},r.a.createElement("div",{className:"container"},i&&i.length>0&&r.a.createElement("div",{className:"row footer__links"},r.a.createElement("div",{className:"col col--5 footer__col"},r.a.createElement("div",{className:"margin-bottom--md"},r.a.createElement(D.a,{className:"navbar__logo",src:"/img/logo-light.svg",alt:"Gnet",width:"150",height:"auto"})),r.a.createElement("div",{className:"margin-bottom--md"},r.a.createElement(M,{description:!1,width:"150px"})),r.a.createElement("div",null,r.a.createElement("a",{href:"https://twitter.com/_andy_pan",target:"_blank"},r.a.createElement("i",{className:"feather icon-twitter",alt:"Gnet's Twitter"})),"\xa0\xa0\xa0\xa0",r.a.createElement("a",{href:"https://github.com/panjf2000/gnet",target:"_blank"},r.a.createElement("i",{className:"feather icon-github",alt:"Gnet's Github Repo"})),"\xa0\xa0\xa0\xa0",r.a.createElement("a",{href:"https://taohuawu.club/rss.xml",target:"_blank"},r.a.createElement("i",{className:"feather icon-rss",alt:"Gnet's RSS feed"})))),i.map((e,t)=>r.a.createElement("div",{key:t,className:"col footer__col"},null!=e.title?r.a.createElement("h4",{className:"footer__title"},e.title):null,null!=e.items&&Array.isArray(e.items)&&e.items.length>0?r.a.createElement("ul",{className:"footer__items"},e.items.map((e,t)=>e.html?r.a.createElement("li",{key:t,className:"footer__item",dangerouslySetInnerHTML:{__html:e.html}}):r.a.createElement("li",{key:e.href||e.to,className:"footer__item"},r.a.createElement(S,e)))):null))),(l||u)&&r.a.createElement("div",{className:"text--center"},l&&l.src&&r.a.createElement("div",{className:"margin-bottom--sm"},l.href?r.a.createElement("a",{href:l.href,target:"_blank",rel:"noopener noreferrer",className:x.a.footerLogoLink},r.a.createElement(B,{alt:l.alt,url:o})):r.a.createElement(B,{alt:l.alt,url:o})),u,r.a.createElement("br",null),r.a.createElement("small",null,r.a.createElement("a",{href:"https://github.com/panjf2000/gnet/security/policy"},"Security Policy"),"\xa0\u2022\xa0",r.a.createElement("a",{href:"https://github.com/panjf2000/gnet/blob/master/PRIVACY.md"},"Privacy Policy")),r.a.createElement("br",null),r.a.createElement("small",null,"Acknowledgement for Timber, Inc.")))):null},L=a(173),H=a(174),I=a(2);a(97);t.a=function(e){const{siteConfig:t={}}=Object(h.a)(),{favicon:a,tagline:n,title:l,themeConfig:{image:o},url:s}=t,{children:c,title:D,noFooter:m,description:d,image:g,keywords:p,permalink:E,version:f}=e,b=D?`${D} | ${l}`:l,y=g||o,F=s+Object(C.a)(y),v=Object(C.a)(a),w=Object(I.h)();let $=w?"https://gnet.host"+(w.pathname.endsWith("/")?w.pathname:w.pathname+"/"):null;return r.a.createElement(H.a,null,r.a.createElement(L.a,null,r.a.createElement(i.a,null,r.a.createElement("html",{lang:"en"}),r.a.createElement("meta",{httpEquiv:"x-ua-compatible",content:"ie=edge"}),b&&r.a.createElement("title",null,b),b&&r.a.createElement("meta",{property:"og:title",content:b}),a&&r.a.createElement("link",{rel:"shortcut icon",href:v}),d&&r.a.createElement("meta",{name:"description",content:d}),d&&r.a.createElement("meta",{property:"og:description",content:d}),f&&r.a.createElement("meta",{name:"docsearch:version",content:f}),p&&p.length&&r.a.createElement("meta",{name:"keywords",content:p.join(",")}),y&&r.a.createElement("meta",{property:"og:image",content:F}),y&&r.a.createElement("meta",{property:"twitter:image",content:F}),y&&r.a.createElement("meta",{name:"twitter:image:alt",content:"Image for "+b}),y&&r.a.createElement("meta",{name:"twitter:site",content:"@vectordotdev"}),y&&r.a.createElement("meta",{name:"twitter:creator",content:"@vectordotdev"}),$&&r.a.createElement("meta",{property:"og:url",content:$}),r.a.createElement("meta",{name:"twitter:card",content:"summary"}),$&&r.a.createElement("link",{rel:"canonical",href:$})),r.a.createElement(u.a,null),r.a.createElement(T,null),r.a.createElement("div",{className:"main-wrapper"},c),!m&&r.a.createElement(P,null)))}},164:function(e,t,a){"use strict";a.d(t,"a",(function(){return u}));var n=a(170),r=a.n(n);function u(e,t){const a=new r.a;return e.map(e=>{let n=e;return"string"==typeof e&&(n={label:e,permalink:`/${t}/tags/${a.slug(e)}`}),function(e,t){if(e.enriched)return e;const a=e.label.split(": ",2),n=a[0],r=a[1];let u="primary";switch(n){case"domain":u="blue";break;case"type":u="pink";break;default:u="primary"}return{category:n,count:e.count,enriched:!0,label:e.label,permalink:e.permalink,style:u,value:r}}(n)})}},170:function(e,t,a){var n=a(176);e.exports=l;var r=Object.hasOwnProperty,u=/\s/g,i=/[\u2000-\u206F\u2E00-\u2E7F\\'!"#$%&()*+,./:;<=>?@[\]^`{|}~\u2019]/g;function l(){if(!(this instanceof l))return new l;this.reset()}function o(e,t){return"string"!=typeof e?"":(t||(e=e.toLowerCase()),e.trim().replace(i,"").replace(n(),"").replace(u,"-"))}l.prototype.slug=function(e,t){for(var a=o(e,!0===t),n=a;r.call(this.occurrences,a);)this.occurrences[n]++,a=n+"-"+this.occurrences[n];return this.occurrences[a]=0,a},l.prototype.reset=function(){this.occurrences=Object.create(null)},l.slug=o},175:function(e,t,a){"use strict";var n=a(0),r=a.n(n),u=a(154),i=a(153),l=a.n(i);t.a=function({count:e,label:t,permalink:a,style:n,value:i,valueOnly:o}){return r.a.createElement(u.a,{to:a+"/",className:l()("badge","badge--rounded","badge--"+n)},o?i:t,e&&r.a.createElement(r.a.Fragment,null," (",e,")"))}},176:function(e,t){e.exports=function(){return/[\xA9\xAE\u203C\u2049\u2122\u2139\u2194-\u2199\u21A9\u21AA\u231A\u231B\u2328\u23CF\u23E9-\u23F3\u23F8-\u23FA\u24C2\u25AA\u25AB\u25B6\u25C0\u25FB-\u25FE\u2600-\u2604\u260E\u2611\u2614\u2615\u2618\u261D\u2620\u2622\u2623\u2626\u262A\u262E\u262F\u2638-\u263A\u2648-\u2653\u2660\u2663\u2665\u2666\u2668\u267B\u267F\u2692-\u2694\u2696\u2697\u2699\u269B\u269C\u26A0\u26A1\u26AA\u26AB\u26B0\u26B1\u26BD\u26BE\u26C4\u26C5\u26C8\u26CE\u26CF\u26D1\u26D3\u26D4\u26E9\u26EA\u26F0-\u26F5\u26F7-\u26FA\u26FD\u2702\u2705\u2708-\u270D\u270F\u2712\u2714\u2716\u271D\u2721\u2728\u2733\u2734\u2744\u2747\u274C\u274E\u2753-\u2755\u2757\u2763\u2764\u2795-\u2797\u27A1\u27B0\u27BF\u2934\u2935\u2B05-\u2B07\u2B1B\u2B1C\u2B50\u2B55\u3030\u303D\u3297\u3299]|\uD83C[\uDC04\uDCCF\uDD70\uDD71\uDD7E\uDD7F\uDD8E\uDD91-\uDD9A\uDE01\uDE02\uDE1A\uDE2F\uDE32-\uDE3A\uDE50\uDE51\uDF00-\uDF21\uDF24-\uDF93\uDF96\uDF97\uDF99-\uDF9B\uDF9E-\uDFF0\uDFF3-\uDFF5\uDFF7-\uDFFF]|\uD83D[\uDC00-\uDCFD\uDCFF-\uDD3D\uDD49-\uDD4E\uDD50-\uDD67\uDD6F\uDD70\uDD73-\uDD79\uDD87\uDD8A-\uDD8D\uDD90\uDD95\uDD96\uDDA5\uDDA8\uDDB1\uDDB2\uDDBC\uDDC2-\uDDC4\uDDD1-\uDDD3\uDDDC-\uDDDE\uDDE1\uDDE3\uDDEF\uDDF3\uDDFA-\uDE4F\uDE80-\uDEC5\uDECB-\uDED0\uDEE0-\uDEE5\uDEE9\uDEEB\uDEEC\uDEF0\uDEF3]|\uD83E[\uDD10-\uDD18\uDD80-\uDD84\uDDC0]|\uD83C\uDDFF\uD83C[\uDDE6\uDDF2\uDDFC]|\uD83C\uDDFE\uD83C[\uDDEA\uDDF9]|\uD83C\uDDFD\uD83C\uDDF0|\uD83C\uDDFC\uD83C[\uDDEB\uDDF8]|\uD83C\uDDFB\uD83C[\uDDE6\uDDE8\uDDEA\uDDEC\uDDEE\uDDF3\uDDFA]|\uD83C\uDDFA\uD83C[\uDDE6\uDDEC\uDDF2\uDDF8\uDDFE\uDDFF]|\uD83C\uDDF9\uD83C[\uDDE6\uDDE8\uDDE9\uDDEB-\uDDED\uDDEF-\uDDF4\uDDF7\uDDF9\uDDFB\uDDFC\uDDFF]|\uD83C\uDDF8\uD83C[\uDDE6-\uDDEA\uDDEC-\uDDF4\uDDF7-\uDDF9\uDDFB\uDDFD-\uDDFF]|\uD83C\uDDF7\uD83C[\uDDEA\uDDF4\uDDF8\uDDFA\uDDFC]|\uD83C\uDDF6\uD83C\uDDE6|\uD83C\uDDF5\uD83C[\uDDE6\uDDEA-\uDDED\uDDF0-\uDDF3\uDDF7-\uDDF9\uDDFC\uDDFE]|\uD83C\uDDF4\uD83C\uDDF2|\uD83C\uDDF3\uD83C[\uDDE6\uDDE8\uDDEA-\uDDEC\uDDEE\uDDF1\uDDF4\uDDF5\uDDF7\uDDFA\uDDFF]|\uD83C\uDDF2\uD83C[\uDDE6\uDDE8-\uDDED\uDDF0-\uDDFF]|\uD83C\uDDF1\uD83C[\uDDE6-\uDDE8\uDDEE\uDDF0\uDDF7-\uDDFB\uDDFE]|\uD83C\uDDF0\uD83C[\uDDEA\uDDEC-\uDDEE\uDDF2\uDDF3\uDDF5\uDDF7\uDDFC\uDDFE\uDDFF]|\uD83C\uDDEF\uD83C[\uDDEA\uDDF2\uDDF4\uDDF5]|\uD83C\uDDEE\uD83C[\uDDE8-\uDDEA\uDDF1-\uDDF4\uDDF6-\uDDF9]|\uD83C\uDDED\uD83C[\uDDF0\uDDF2\uDDF3\uDDF7\uDDF9\uDDFA]|\uD83C\uDDEC\uD83C[\uDDE6\uDDE7\uDDE9-\uDDEE\uDDF1-\uDDF3\uDDF5-\uDDFA\uDDFC\uDDFE]|\uD83C\uDDEB\uD83C[\uDDEE-\uDDF0\uDDF2\uDDF4\uDDF7]|\uD83C\uDDEA\uD83C[\uDDE6\uDDE8\uDDEA\uDDEC\uDDED\uDDF7-\uDDFA]|\uD83C\uDDE9\uD83C[\uDDEA\uDDEC\uDDEF\uDDF0\uDDF2\uDDF4\uDDFF]|\uD83C\uDDE8\uD83C[\uDDE6\uDDE8\uDDE9\uDDEB-\uDDEE\uDDF0-\uDDF5\uDDF7\uDDFA-\uDDFF]|\uD83C\uDDE7\uD83C[\uDDE6\uDDE7\uDDE9-\uDDEF\uDDF1-\uDDF4\uDDF6-\uDDF9\uDDFB\uDDFC\uDDFE\uDDFF]|\uD83C\uDDE6\uD83C[\uDDE8-\uDDEC\uDDEE\uDDF1\uDDF2\uDDF4\uDDF6-\uDDFA\uDDFC\uDDFD\uDDFF]|[#\*0-9]\u20E3/g}},181:function(e,t,a){"use strict";var n=a(0),r=a.n(n),u=a(153),i=a.n(u),l=a(152);a(101);t.a=function({bio:e,className:t,github:a,nameSuffix:n,rel:u,size:o,subTitle:s,vertical:c}){const D=Object(l.a)(),{siteConfig:m={}}=D,{metadata:{team:d}}=m.customFields,h=d.find(e=>e.github==a)||d.find(e=>"ben"==e.id);return r.a.createElement("div",{className:i()("avatar",t,{["avatar--"+o]:o,"avatar--vertical":c})},r.a.createElement("img",{className:i()("avatar__photo","avatar__photo--"+o),src:h.avatar}),r.a.createElement("div",{className:"avatar__intro"},r.a.createElement("div",{className:"avatar__name"},r.a.createElement("a",{href:h.github,target:"_blank",rel:u},h.name),n),s&&r.a.createElement("small",{className:"avatar__subtitle"},s),!s&&e&&r.a.createElement("small",{className:"avatar__subtitle",dangerouslySetInnerHTML:{__html:h.bio}})))}},182:function(e,t,a){"use strict";var n=a(1),r=a(0),u=a.n(r),i=(a(154),a(175)),l=a(153),o=a.n(l),s=a(164),c=a(102),D=a.n(c);t.a=function({block:e,colorProfile:t,tags:a,valuesOnly:r}){const l=Object(s.a)(a,t);return u.a.createElement("span",{className:o()(D.a.tags,{[D.a.tagsBlock]:e})},l.map((e,t)=>u.a.createElement(i.a,Object(n.a)({key:t,valueOnly:r},e))))}},183:function(e,t,a){var n;!function(r){"use strict";var u,i,l,o=(u=/d{1,4}|m{1,4}|yy(?:yy)?|([HhMsTt])\1?|[LloSZWN]|"[^"]*"|'[^']*'/g,i=/\b(?:[PMCEA][SDP]T|(?:Pacific|Mountain|Central|Eastern|Atlantic) (?:Standard|Daylight|Prevailing) Time|(?:GMT|UTC)(?:[-+]\d{4})?)\b/g,l=/[^-+\dA-Z]/g,function(e,t,a,n){if(1!==arguments.length||"string"!==m(e)||/\d/.test(e)||(t=e,e=void 0),(e=e||new Date)instanceof Date||(e=new Date(e)),isNaN(e))throw TypeError("Invalid date");var r=(t=String(o.masks[t]||t||o.masks.default)).slice(0,4);"UTC:"!==r&&"GMT:"!==r||(t=t.slice(4),a=!0,"GMT:"===r&&(n=!0));var d=a?"getUTC":"get",h=e[d+"Date"](),g=e[d+"Day"](),p=e[d+"Month"](),E=e[d+"FullYear"](),f=e[d+"Hours"](),b=e[d+"Minutes"](),y=e[d+"Seconds"](),F=e[d+"Milliseconds"](),v=a?0:e.getTimezoneOffset(),C=c(e),w=D(e),$={d:h,dd:s(h),ddd:o.i18n.dayNames[g],dddd:o.i18n.dayNames[g+7],m:p+1,mm:s(p+1),mmm:o.i18n.monthNames[p],mmmm:o.i18n.monthNames[p+12],yy:String(E).slice(2),yyyy:E,h:f%12||12,hh:s(f%12||12),H:f,HH:s(f),M:b,MM:s(b),s:y,ss:s(y),l:s(F,3),L:s(Math.round(F/10)),t:f<12?o.i18n.timeNames[0]:o.i18n.timeNames[1],tt:f<12?o.i18n.timeNames[2]:o.i18n.timeNames[3],T:f<12?o.i18n.timeNames[4]:o.i18n.timeNames[5],TT:f<12?o.i18n.timeNames[6]:o.i18n.timeNames[7],Z:n?"GMT":a?"UTC":(String(e).match(i)||[""]).pop().replace(l,""),o:(v>0?"-":"+")+s(100*Math.floor(Math.abs(v)/60)+Math.abs(v)%60,4),S:["th","st","nd","rd"][h%10>3?0:(h%100-h%10!=10)*h%10],W:C,N:w};return t.replace(u,(function(e){return e in $?$[e]:e.slice(1,e.length-1)}))});function s(e,t){for(e=String(e),t=t||2;e.length<t;)e="0"+e;return e}function c(e){var t=new Date(e.getFullYear(),e.getMonth(),e.getDate());t.setDate(t.getDate()-(t.getDay()+6)%7+3);var a=new Date(t.getFullYear(),0,4);a.setDate(a.getDate()-(a.getDay()+6)%7+3);var n=t.getTimezoneOffset()-a.getTimezoneOffset();t.setHours(t.getHours()-n);var r=(t-a)/6048e5;return 1+Math.floor(r)}function D(e){var t=e.getDay();return 0===t&&(t=7),t}function m(e){return null===e?"null":void 0===e?"undefined":"object"!=typeof e?typeof e:Array.isArray(e)?"array":{}.toString.call(e).slice(8,-1).toLowerCase()}o.masks={default:"ddd mmm dd yyyy HH:MM:ss",shortDate:"m/d/yy",mediumDate:"mmm d, yyyy",longDate:"mmmm d, yyyy",fullDate:"dddd, mmmm d, yyyy",shortTime:"h:MM TT",mediumTime:"h:MM:ss TT",longTime:"h:MM:ss TT Z",isoDate:"yyyy-mm-dd",isoTime:"HH:MM:ss",isoDateTime:"yyyy-mm-dd'T'HH:MM:sso",isoUtcDateTime:"UTC:yyyy-mm-dd'T'HH:MM:ss'Z'",expiresHeaderFormat:"ddd, dd mmm yyyy HH:MM:ss Z"},o.i18n={dayNames:["Sun","Mon","Tue","Wed","Thu","Fri","Sat","Sunday","Monday","Tuesday","Wednesday","Thursday","Friday","Saturday"],monthNames:["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec","January","February","March","April","May","June","July","August","September","October","November","December"],timeNames:["a","p","am","pm","A","P","AM","PM"]},void 0===(n=function(){return o}.call(t,a,t,e))||(e.exports=n)}()},184:function(e,t,a){e.exports=function(){var e=[],t=[],a={},n={},r={};function u(e){return"string"==typeof e?new RegExp("^"+e+"$","i"):e}function i(e,t){return e===t?t:e===e.toLowerCase()?t.toLowerCase():e===e.toUpperCase()?t.toUpperCase():e[0]===e[0].toUpperCase()?t.charAt(0).toUpperCase()+t.substr(1).toLowerCase():t.toLowerCase()}function l(e,t){return e.replace(/\$(\d{1,2})/g,(function(e,a){return t[a]||""}))}function o(e,t){return e.replace(t[0],(function(a,n){var r=l(t[1],arguments);return i(""===a?e[n-1]:a,r)}))}function s(e,t,n){if(!e.length||a.hasOwnProperty(e))return t;for(var r=n.length;r--;){var u=n[r];if(u[0].test(t))return o(t,u)}return t}function c(e,t,a){return function(n){var r=n.toLowerCase();return t.hasOwnProperty(r)?i(n,r):e.hasOwnProperty(r)?i(n,e[r]):s(r,n,a)}}function D(e,t,a,n){return function(n){var r=n.toLowerCase();return!!t.hasOwnProperty(r)||!e.hasOwnProperty(r)&&s(r,r,a)===r}}function m(e,t,a){return(a?t+" ":"")+(1===t?m.singular(e):m.plural(e))}return m.plural=c(r,n,e),m.isPlural=D(r,n,e),m.singular=c(n,r,t),m.isSingular=D(n,r,t),m.addPluralRule=function(t,a){e.push([u(t),a])},m.addSingularRule=function(e,a){t.push([u(e),a])},m.addUncountableRule=function(e){"string"!=typeof e?(m.addPluralRule(e,"$0"),m.addSingularRule(e,"$0")):a[e.toLowerCase()]=!0},m.addIrregularRule=function(e,t){t=t.toLowerCase(),e=e.toLowerCase(),r[e]=t,n[t]=e},[["I","we"],["me","us"],["he","they"],["she","they"],["them","them"],["myself","ourselves"],["yourself","yourselves"],["itself","themselves"],["herself","themselves"],["himself","themselves"],["themself","themselves"],["is","are"],["was","were"],["has","have"],["this","these"],["that","those"],["echo","echoes"],["dingo","dingoes"],["volcano","volcanoes"],["tornado","tornadoes"],["torpedo","torpedoes"],["genus","genera"],["viscus","viscera"],["stigma","stigmata"],["stoma","stomata"],["dogma","dogmata"],["lemma","lemmata"],["schema","schemata"],["anathema","anathemata"],["ox","oxen"],["axe","axes"],["die","dice"],["yes","yeses"],["foot","feet"],["eave","eaves"],["goose","geese"],["tooth","teeth"],["quiz","quizzes"],["human","humans"],["proof","proofs"],["carve","carves"],["valve","valves"],["looey","looies"],["thief","thieves"],["groove","grooves"],["pickaxe","pickaxes"],["passerby","passersby"]].forEach((function(e){return m.addIrregularRule(e[0],e[1])})),[[/s?$/i,"s"],[/[^\u0000-\u007F]$/i,"$0"],[/([^aeiou]ese)$/i,"$1"],[/(ax|test)is$/i,"$1es"],[/(alias|[^aou]us|t[lm]as|gas|ris)$/i,"$1es"],[/(e[mn]u)s?$/i,"$1s"],[/([^l]ias|[aeiou]las|[ejzr]as|[iu]am)$/i,"$1"],[/(alumn|syllab|vir|radi|nucle|fung|cact|stimul|termin|bacill|foc|uter|loc|strat)(?:us|i)$/i,"$1i"],[/(alumn|alg|vertebr)(?:a|ae)$/i,"$1ae"],[/(seraph|cherub)(?:im)?$/i,"$1im"],[/(her|at|gr)o$/i,"$1oes"],[/(agend|addend|millenni|dat|extrem|bacteri|desiderat|strat|candelabr|errat|ov|symposi|curricul|automat|quor)(?:a|um)$/i,"$1a"],[/(apheli|hyperbat|periheli|asyndet|noumen|phenomen|criteri|organ|prolegomen|hedr|automat)(?:a|on)$/i,"$1a"],[/sis$/i,"ses"],[/(?:(kni|wi|li)fe|(ar|l|ea|eo|oa|hoo)f)$/i,"$1$2ves"],[/([^aeiouy]|qu)y$/i,"$1ies"],[/([^ch][ieo][ln])ey$/i,"$1ies"],[/(x|ch|ss|sh|zz)$/i,"$1es"],[/(matr|cod|mur|sil|vert|ind|append)(?:ix|ex)$/i,"$1ices"],[/\b((?:tit)?m|l)(?:ice|ouse)$/i,"$1ice"],[/(pe)(?:rson|ople)$/i,"$1ople"],[/(child)(?:ren)?$/i,"$1ren"],[/eaux$/i,"$0"],[/m[ae]n$/i,"men"],["thou","you"]].forEach((function(e){return m.addPluralRule(e[0],e[1])})),[[/s$/i,""],[/(ss)$/i,"$1"],[/(wi|kni|(?:after|half|high|low|mid|non|night|[^\w]|^)li)ves$/i,"$1fe"],[/(ar|(?:wo|[ae])l|[eo][ao])ves$/i,"$1f"],[/ies$/i,"y"],[/\b([pl]|zomb|(?:neck|cross)?t|coll|faer|food|gen|goon|group|lass|talk|goal|cut)ies$/i,"$1ie"],[/\b(mon|smil)ies$/i,"$1ey"],[/\b((?:tit)?m|l)ice$/i,"$1ouse"],[/(seraph|cherub)im$/i,"$1"],[/(x|ch|ss|sh|zz|tto|go|cho|alias|[^aou]us|t[lm]as|gas|(?:her|at|gr)o|[aeiou]ris)(?:es)?$/i,"$1"],[/(analy|diagno|parenthe|progno|synop|the|empha|cri|ne)(?:sis|ses)$/i,"$1sis"],[/(movie|twelve|abuse|e[mn]u)s$/i,"$1"],[/(test)(?:is|es)$/i,"$1is"],[/(alumn|syllab|vir|radi|nucle|fung|cact|stimul|termin|bacill|foc|uter|loc|strat)(?:us|i)$/i,"$1us"],[/(agend|addend|millenni|dat|extrem|bacteri|desiderat|strat|candelabr|errat|ov|symposi|curricul|quor)a$/i,"$1um"],[/(apheli|hyperbat|periheli|asyndet|noumen|phenomen|criteri|organ|prolegomen|hedr|automat)a$/i,"$1on"],[/(alumn|alg|vertebr)ae$/i,"$1a"],[/(cod|mur|sil|vert|ind)ices$/i,"$1ex"],[/(matr|append)ices$/i,"$1ix"],[/(pe)(rson|ople)$/i,"$1rson"],[/(child)ren$/i,"$1"],[/(eau)x?$/i,"$1"],[/men$/i,"man"]].forEach((function(e){return m.addSingularRule(e[0],e[1])})),["adulthood","advice","agenda","aid","aircraft","alcohol","ammo","analytics","anime","athletics","audio","bison","blood","bream","buffalo","butter","carp","cash","chassis","chess","clothing","cod","commerce","cooperation","corps","debris","diabetes","digestion","elk","energy","equipment","excretion","expertise","firmware","flounder","fun","gallows","garbage","graffiti","hardware","headquarters","health","herpes","highjinks","homework","housework","information","jeans","justice","kudos","labour","literature","machinery","mackerel","mail","media","mews","moose","music","mud","manga","news","only","personnel","pike","plankton","pliers","police","pollution","premises","rain","research","rice","salmon","scissors","series","sewage","shambles","shrimp","software","species","staff","swine","tennis","traffic","transportation","trout","tuna","wealth","welfare","whiting","wildebeest","wildlife","you",/pok[e\xe9]mon$/i,/[^aeiou]ese$/i,/deer$/i,/fish$/i,/measles$/i,/o[iu]s$/i,/pox$/i,/sheep$/i].forEach(m.addUncountableRule),m}()},185:function(e,t,a){"use strict";var n=a(0),r=a.n(n),u=["second","minute","hour","day","week","month","year"],i=["\u79d2","\u5206\u949f","\u5c0f\u65f6","\u5929","\u5468","\u4e2a\u6708","\u5e74"],l={},o=function(e,t){l[e]=t},s=function(e){return l[e]||l.en_US},c=[60,60,24,7,365/7/12,12];function D(e){return e instanceof Date?e:!isNaN(e)||/^\d+$/.test(e)?new Date(parseInt(e)):(e=(e||"").trim().replace(/\.\d+/,"").replace(/-/,"/").replace(/-/,"/").replace(/(\d)T(\d)/,"$1 $2").replace(/Z/," UTC").replace(/([+-]\d\d):?(\d\d)/," $1$2"),new Date(e))}function m(e,t){for(var a=e<0?1:0,n=e=Math.abs(e),r=0;e>=c[r]&&r<c.length;r++)e/=c[r];return(e=Math.floor(e))>(0===(r*=2)?9:1)&&(r+=1),t(e,r,n)[a].replace("%s",e.toString())}function d(e,t){return(+(t?D(t):new Date)-+D(e))/1e3}function h(e){return parseInt(e.getAttribute("timeago-id"))}var g={},p=function(e){clearTimeout(e),delete g[e]};function E(e,t,a,n){p(h(e));var r=n.relativeDate,u=n.minInterval,i=d(t,r);e.innerText=m(i,a);var l=setTimeout((function(){E(e,t,a,n)}),Math.min(1e3*Math.max(function(e){for(var t=1,a=0,n=Math.abs(e);e>=c[a]&&a<c.length;a++)e/=c[a],t*=c[a];return n=(n%=t)?t-n:t,Math.ceil(n)}(i),u||1),2147483647));g[l]=0,function(e,t){e.setAttribute("timeago-id",t)}(e,l)}function f(e){e?p(h(e)):Object.keys(g).forEach(p)}o("en_US",(function(e,t){if(0===t)return["just now","right now"];var a=u[Math.floor(t/2)];return e>1&&(a+="s"),[e+" "+a+" ago","in "+e+" "+a]})),o("zh_CN",(function(e,t){if(0===t)return["\u521a\u521a","\u7247\u523b\u540e"];var a=i[~~(t/2)];return[e+" "+a+"\u524d",e+" "+a+"\u540e"]}));var b,y=(b=function(e,t){return(b=Object.setPrototypeOf||{__proto__:[]}instanceof Array&&function(e,t){e.__proto__=t}||function(e,t){for(var a in t)t.hasOwnProperty(a)&&(e[a]=t[a])})(e,t)},function(e,t){function a(){this.constructor=e}b(e,t),e.prototype=null===t?Object.create(t):(a.prototype=t.prototype,new a)}),F=function(){return(F=Object.assign||function(e){for(var t,a=1,n=arguments.length;a<n;a++)for(var r in t=arguments[a])Object.prototype.hasOwnProperty.call(t,r)&&(e[r]=t[r]);return e}).apply(this,arguments)},v=function(e,t){var a={};for(var n in e)Object.prototype.hasOwnProperty.call(e,n)&&t.indexOf(n)<0&&(a[n]=e[n]);if(null!=e&&"function"==typeof Object.getOwnPropertySymbols){var r=0;for(n=Object.getOwnPropertySymbols(e);r<n.length;r++)t.indexOf(n[r])<0&&Object.prototype.propertyIsEnumerable.call(e,n[r])&&(a[n[r]]=e[n[r]])}return a},C=function(e){function t(){var t=null!==e&&e.apply(this,arguments)||this;return t.dom=null,t}return y(t,e),t.prototype.componentDidMount=function(){this.renderTimeAgo()},t.prototype.componentDidUpdate=function(){this.renderTimeAgo()},t.prototype.renderTimeAgo=function(){var e,t=this.props,a=t.live,n=t.datetime,r=t.locale,u=t.opts;f(this.dom),!1!==a&&(this.dom.setAttribute("datetime",""+((e=n)instanceof Date?e.getTime():e)),function(e,t,a){var n=e.length?e:[e];n.forEach((function(e){E(e,function(e){return e.getAttribute("datetime")}(e),s(t),a||{})}))}(this.dom,r,u))},t.prototype.componentWillUnmount=function(){f(this.dom)},t.prototype.render=function(){var e=this,t=this.props,a=t.datetime,n=(t.live,t.locale),u=t.opts,i=v(t,["datetime","live","locale","opts"]);return r.a.createElement("time",F({ref:function(t){e.dom=t}},i),function(e,t,a){return m(d(e,a&&a.relativeDate),s(t))}(a,n,u))},t.defaultProps={live:!0,className:""},t}(r.a.Component);t.a=C},204:function(e,t,a){"use strict";const n=e=>{if("string"!=typeof e)throw new TypeError("Expected a string");return e.toLowerCase().replace(/(?:^|\s|-)\S/g,e=>e.toUpperCase())};e.exports=n,e.exports.default=n},208:function(e,t,a){"use strict";var n=a(1),r=(a(12),a(0)),u=a.n(r),i=a(169),l=a(181),o=a(154),s=a(182),c=a(185),D=(a(197),a(153)),m=a.n(D),d=a(183),h=a.n(d),g=a(164);var p=function({authorGithub:e,colorize:t,dateString:a,description:n,headingDepth:r,hideAuthor:i,hideTags:D,permalink:d,prNumbers:p,tags:E,title:f}){const b=new Date(Date.parse(a)),y=h()(b,"mmm dS, yyyy");let F=Object(g.a)(E,"highlights");F=F.concat(p.map(e=>({enriched:!0,label:u.a.createElement(u.a.Fragment,null,u.a.createElement("i",{className:"feather icon-git-pull-request"})," ",e),permalink:"https://github.com/panjf2000/gnet/commit/"+e,style:"secondary"})));const v=F.find(e=>"domain"==e.category),C=v?v.value:null,w=F.find(e=>"type"==e.category),$=w?w.value:null,k="h"+(r||3);let N=null;if(t)switch($){case"breaking change":N="danger";break;case"enhancement":N="pink";break;case"new feature":N="primary";break;case"performance":N="warning"}const _=u.a.createElement(u.a.Fragment,null,u.a.createElement("span",{className:"time"},u.a.createElement("span",{className:"formatted-time"},y),u.a.createElement("span",{className:"separator"}," / "),u.a.createElement(c.a,{title:y,pubdate:"pubdate",datetime:b})),u.a.createElement("span",{className:"separator"}," / "),u.a.createElement("span",{className:"author-title"},"Gnet creator"));return u.a.createElement(o.a,{to:d,className:m()("panel","panel--"+N,"domain-bg","domain-bg--hover","domain-bg--"+C)},u.a.createElement("article",null,u.a.createElement(k,null,f),u.a.createElement("div",{className:"subtitle"},n),!i&&e&&u.a.createElement(l.a,{github:e,size:"sm",subTitle:_,rel:"author"}),!D&&F.length>0&&u.a.createElement("div",null,u.a.createElement(s.a,{colorProfile:"blog",tags:F}))))},E=a(170),f=a.n(E),b=a(184),y=a.n(b),F=a(204),v=a.n(F);Object(i.a)("h2");const C=Object(i.a)("h3");function w({groupBy:e,group:t}){const a=new f.a;switch(e){case"release":return u.a.createElement("li",{className:"header sticky"},u.a.createElement("h3",null,u.a.createElement(o.a,{to:`/releases/${t}/`},v()(t))));case"type":let n=null,r=y()(v()(t)),i=null;switch(t){case"breaking change":n="alert-triangle",i="danger";break;case"enhancement":n="arrow-up-circle",i="pink";break;case"new feature":n="gift",i="primary";break;case"performance":n="zap",r="Performance Improvements",i="warning"}return u.a.createElement("li",{className:"header sticky"},u.a.createElement(C,{id:a.slug(t+"-highlights"),className:"text--"+i},n&&u.a.createElement("i",{className:"feather icon-"+n})," ",r));default:throw Error("unknown group: "+e)}}t.a=function({author:e,clean:t,colorize:a,groupBy:r,items:i,tags:l,timeline:o}){let s=r||"release",c=function(e){return e.map(e=>{if(e.content){const{content:t}=e,{frontMatter:a,metadata:n}=t,{author_github:r,pr_numbers:u,release:i,title:l}=a,{date:o,description:s,permalink:c,tags:D}=n;let m={};return m.authorGithub=r,m.dateString=o,m.description=s,m.permalink=c,m.prNumbers=u,m.release=i,m.tags=D,m.title=l,m}return e})}(i),D=_.groupBy(c,s),d=!1!==o?Object.keys(D):Object.keys(D).sort();return u.a.createElement("ul",{className:m()("connected-list","connected-list--clean")},d.map((t,r)=>{let i=D[t];return u.a.createElement(u.a.Fragment,null,u.a.createElement(w,{groupBy:s,group:t}),u.a.createElement("ul",{className:m()("connected-list",{"connected-list--timeline":!1!==o})},i.map((t,r)=>u.a.createElement("li",{key:r},u.a.createElement(p,Object(n.a)({},t,{colorize:a,hideAuthor:0==e,hideTags:0==l}))))))}))}}}]);