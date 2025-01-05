(window.webpackJsonp=window.webpackJsonp||[]).push([[27],{170:function(e,n,t){"use strict";t.r(n),t.d(n,"frontMatter",(function(){return o})),t.d(n,"metadata",(function(){return i})),t.d(n,"rightToc",(function(){return c})),t.d(n,"default",(function(){return b}));var r=t(1),a=t(9),l=(t(0),t(270)),o={last_modified_on:"2025-01-04",id:"announcing-gnet-v2-7-0",title:"Announcing gnet v2.7.0",description:"Hello World! We present you, gnet v2.7.0!",author_github:"https://github.com/panjf2000",tags:["type: announcement","domain: presentation"]},i={permalink:"/blog/announcing-gnet-v2-7-0",source:"@site/blog/2025-01-04-announcing-gnet-v2-7-0.md",description:"Hello World! We present you, gnet v2.7.0!",date:"2025-01-04T00:00:00.000Z",tags:[{label:"type: announcement",permalink:"/blog/tags/type-announcement"},{label:"domain: presentation",permalink:"/blog/tags/domain-presentation"}],title:"Announcing gnet v2.7.0",readingTime:1.78,truncated:!1,nextItem:{title:"Announcing gnet v2.6.0",permalink:"/blog/announcing-gnet-v2-6-0"}},c=[],p={rightToc:c};function b(e){var n=e.components,t=Object(a.a)(e,["components"]);return Object(l.b)("wrapper",Object(r.a)({},p,t,{components:n,mdxType:"MDXLayout"}),Object(l.b)("p",null,Object(l.b)("img",Object(r.a)({parentName:"p"},{src:"/img/gnet-v2-7-0.jpg",alt:null}))),Object(l.b)("p",null,"The ",Object(l.b)("inlineCode",{parentName:"p"},"gnet")," v2.7.0 is officially released!"),Object(l.b)("p",null,Object(l.b)("strong",{parentName:"p"},Object(l.b)("em",{parentName:"strong"},"In this release, most of the core internal packages used by gnet are now available outside of gnet!"))),Object(l.b)("p",null,"Take ",Object(l.b)("inlineCode",{parentName:"p"},"netpoll")," package as an example:"),Object(l.b)("p",null,"Package ",Object(l.b)("inlineCode",{parentName:"p"},"netpoll")," provides a portable event-driven interface for network I/O."),Object(l.b)("p",null,"The underlying facility of event notification is OS-specific:"),Object(l.b)("ul",null,Object(l.b)("li",{parentName:"ul"},Object(l.b)("a",Object(r.a)({parentName:"li"},{href:"https://man7.org/linux/man-pages/man7/epoll.7.html"}),"epoll")," on Linux"),Object(l.b)("li",{parentName:"ul"},Object(l.b)("a",Object(r.a)({parentName:"li"},{href:"https://man.freebsd.org/cgi/man.cgi?kqueue"}),"kqueue")," on *BSD/Darwin")),Object(l.b)("p",null,"With the help of the ",Object(l.b)("inlineCode",{parentName:"p"},"netpoll")," package, you can easily build your own high-performance\nevent-driven network applications based on epoll/kqueue."),Object(l.b)("p",null,"The ",Object(l.b)("inlineCode",{parentName:"p"},"Poller")," represents the event notification facility whose backend is epoll or kqueue.\nThe ",Object(l.b)("inlineCode",{parentName:"p"},"OpenPoller")," function creates a new ",Object(l.b)("inlineCode",{parentName:"p"},"Poller")," instance:"),Object(l.b)("pre",null,Object(l.b)("code",Object(r.a)({parentName:"pre"},{className:"language-go"}),'    poller, err := netpoll.OpenPoller()\n    if err != nil {\n        // handle error\n    }\n\n    defer poller.Close()\n\n    addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:9090")\n    if err != nil {\n        // handle error\n    }\n    c, err := net.DialTCP("tcp", nil, addr)\n    if err != nil {\n        // handle error\n    }\n\n    f, err := c.File()\n    if err != nil {\n        // handle error\n    }\n\n    closeClient := func() {\n        c.Close()\n        f.Close()\n    }\n    defer closeClient()\n')),Object(l.b)("p",null,"The ",Object(l.b)("inlineCode",{parentName:"p"},"PollAttachment")," consists of a file descriptor and its callback function.\n",Object(l.b)("inlineCode",{parentName:"p"},"PollAttachment")," is used to register a file descriptor to ",Object(l.b)("inlineCode",{parentName:"p"},"Poller"),".\nThe callback function is called when an event occurs on the file descriptor:"),Object(l.b)("pre",null,Object(l.b)("code",Object(r.a)({parentName:"pre"},{className:"language-go"}),'    pa := netpoll.PollAttachment{\n        FD: int(f.Fd()),\n        Callback: func(fd int, event netpoll.IOEvent, flags netpoll.IOFlags) error {\n            if netpoll.IsErrorEvent(event, flags) {\n                closeClient()\n                return errors.ErrEngineShutdown\n            }\n\n            if netpoll.IsReadEvent(event) {\n                buf := make([]byte, 64)\n                // Read data from the connection.\n                _, err := c.Read(buf)\n                if err != nil {\n                    closeClient()\n                    return errors.ErrEngineShutdown\n                }\n                // Process the data...\n            }\n\n            if netpoll.IsWriteEvent(event) {\n                // Write data to the connection.\n                _, err := c.Write([]byte("hello"))\n                if err != nil {\n                    closeClient()\n                    return errors.ErrEngineShutdown\n                }\n            }\n\n            return nil\n        }}\n\n    if err := poller.AddReadWrite(&pa, false); err != nil {\n        // handle error\n    }\n')),Object(l.b)("p",null,"The ",Object(l.b)("inlineCode",{parentName:"p"},"Poller.Polling")," function starts the event loop monitoring file descriptors and\nwaiting for I/O events to occur:"),Object(l.b)("pre",null,Object(l.b)("code",Object(r.a)({parentName:"pre"},{className:"language-go"}),"    poller.Polling(func(fd int, event netpoll.IOEvent, flags netpoll.IOFlags) error {\n        return pa.Callback(fd, event, flags)\n    })\n")),Object(l.b)("p",null,"Or"),Object(l.b)("pre",null,Object(l.b)("code",Object(r.a)({parentName:"pre"},{className:"language-go"}),"    poller.Polling()\n")),Object(l.b)("p",null,"if you've enabled the build tag ",Object(l.b)("inlineCode",{parentName:"p"},"poll_opt"),"."),Object(l.b)("p",null,"Check out ",Object(l.b)("a",Object(r.a)({parentName:"p"},{href:"https://pkg.go.dev/github.com/panjf2000/gnet/v2@v2.7.0/pkg"}),"gnet/pkg")," for more details."),Object(l.b)("p",null,"P.S. Follow me on Twitter ",Object(l.b)("a",Object(r.a)({parentName:"p"},{href:"https://twitter.com/panjf2000"}),"@panjf2000")," to get the latest updates about gnet!"))}b.isMDXComponent=!0},270:function(e,n,t){"use strict";t.d(n,"a",(function(){return u})),t.d(n,"b",(function(){return d}));var r=t(0),a=t.n(r);function l(e,n,t){return n in e?Object.defineProperty(e,n,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[n]=t,e}function o(e,n){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);n&&(r=r.filter((function(n){return Object.getOwnPropertyDescriptor(e,n).enumerable}))),t.push.apply(t,r)}return t}function i(e){for(var n=1;n<arguments.length;n++){var t=null!=arguments[n]?arguments[n]:{};n%2?o(Object(t),!0).forEach((function(n){l(e,n,t[n])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):o(Object(t)).forEach((function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))}))}return e}function c(e,n){if(null==e)return{};var t,r,a=function(e,n){if(null==e)return{};var t,r,a={},l=Object.keys(e);for(r=0;r<l.length;r++)t=l[r],n.indexOf(t)>=0||(a[t]=e[t]);return a}(e,n);if(Object.getOwnPropertySymbols){var l=Object.getOwnPropertySymbols(e);for(r=0;r<l.length;r++)t=l[r],n.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(a[t]=e[t])}return a}var p=a.a.createContext({}),b=function(e){var n=a.a.useContext(p),t=n;return e&&(t="function"==typeof e?e(n):i({},n,{},e)),t},u=function(e){var n=b(e.components);return a.a.createElement(p.Provider,{value:n},e.children)},s={inlineCode:"code",wrapper:function(e){var n=e.children;return a.a.createElement(a.a.Fragment,{},n)}},f=Object(r.forwardRef)((function(e,n){var t=e.components,r=e.mdxType,l=e.originalType,o=e.parentName,p=c(e,["components","mdxType","originalType","parentName"]),u=b(t),f=r,d=u["".concat(o,".").concat(f)]||u[f]||s[f]||l;return t?a.a.createElement(d,i({ref:n},p,{components:t})):a.a.createElement(d,i({ref:n},p))}));function d(e,n){var t=arguments,r=n&&n.mdxType;if("string"==typeof e||r){var l=t.length,o=new Array(l);o[0]=f;var i={};for(var c in n)hasOwnProperty.call(n,c)&&(i[c]=n[c]);i.originalType=e,i.mdxType="string"==typeof e?e:r,o[1]=i;for(var p=2;p<l;p++)o[p]=t[p];return a.a.createElement.apply(null,o)}return a.a.createElement.apply(null,t)}f.displayName="MDXCreateElement"}}]);