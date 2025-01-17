import React, { useState, useEffect } from 'react';

import Heading from '@theme/Heading';
import Layout from '@theme/Layout';
import CodeBlock from "@theme/CodeBlock";
import Link from '@docusaurus/Link';
import SVG from 'react-inlinesvg';
import TabItem from '@theme/TabItem';
import Tabs from '@theme/Tabs';

import classnames from 'classnames';
import { fetchNewHighlight } from '@site/src/exports/newHighlight';
import {fetchNewPost} from '@site/src/exports/newPost';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';

import _ from 'lodash';
import styles from './index.module.css';

import './index.css';

const AnchoredH2 = Heading('h2');

const features = [
  {
    title: 'Ultra-Fast',
    icon: 'zap',
    description: (
      <>
        Built in <a href="https://go.dev/">Go</a>, gnet is an <a href="#performance">ultra-fast and
        memory-efficient</a> networking framework. It is built from scratch by exploiting the event-driven techniques -
        <a href="https://man7.org/linux/man-pages/man7/epoll.7.html">epoll</a> and <a href="https://man.freebsd.org/cgi/man.cgi?kqueue">kqueue</a>.
        gnet sells itself as a high-performance, lightweight, non-blocking, event-driven networking framework
        written in pure Go which works on the transport layer with TCP/UDP protocols and Unix Domain Socket.
      </>
    ),
  },
  {
    title: 'Lock-Free',
    icon: 'unlock',
    description: (
      <>
        gnet is lock-free during the entire runtime, which keeps gnet free from synchronization issues and speeds it up.
      </>
    ),
  },
  {
    title: 'Concise & Easy-to-use APIs',
    icon: 'layers',
    description: (
      <>
        gnet provides concise and easy-to-use APIs for users, it provides only the essential APIs and takes over most of the tough work for developers,
        minimizing the complexity of networking applications so that developers are able to concentrate on business logic instead of the underlying
        implementations of networking.
      </>
    ),
  },
  {
    title: 'Multiple Protocols',
    icon: 'grid',
    description: (
      <>
        gnet supports multiple protocols/IPC mechanism: TCP, UDP and Unix Domain Socket, enabling you to develop
        a variety of networking applications.
      </>
    ),
  },
  {
    title: 'Cross Platform',
    icon: 'cpu',
    description: (
      <>
        gnet is devised as a cross-platform framework, as a result, it works faultlessly on multiple platforms: Linux, FreeBSD, DragonFly BSD, Windows.
      </>
    ),
  },
  {
    title: 'Powerful Libraries',
    icon: 'briefcase',
    description: (
      <>
        There is a rich set of libraries in gnet, such as memory pool, goroutine pool, elastic buffers, logging package, etc.,
        which makes it convenient for developers to build fast and efficient networking applications.
      </>
    ),
  },
];

function Features({ features }) {
  let rows = [];

  let i, j, temparray, chunk = 3;
  for (i = 0, j = features.length; i < j; i += chunk) {
    let featuresChunk = features.slice(i, i + chunk);

    rows.push(
      <div key={`features${i}`} className="row">
        {featuresChunk.map((props, idx) => (
          <Feature key={idx} {...props} />
        ))}
      </div>
    );
  }

  return (
    <section className={styles.features}>
      <div className="container">
        <AnchoredH2 id="features">Why gnet?</AnchoredH2>
        {rows}
      </div>
    </section>
  );
}

function Feature({ icon, title, description }) {
  return (
    <div className={classnames('col col--4', styles.feature)}>
      <div className={styles.featureIcon}>
        <i className={classnames('feather', `icon-${icon}`)}></i>
      </div>
      <h3>{title}</h3>
      <p>{description}</p>
    </div>
  );
}

function Topologies() {
  return (
    <section className="topologies">
      <div className="container">
        <AnchoredH2 id="topologies">Networking model of multiple reactors</AnchoredH2>
        <div className="sub-title">Learn how gnet works</div>

        <Tabs
          centered={true}
          className="rounded"
          defaultValue="multi-reactors"
          values={[
            { label: <><i className="feather icon-box"></i>&nbsp;Multiple Reactors</>, value: 'multi-reactors', },
            { label: <><i className="feather icon-codepen"></i>&nbsp;Multiple Reactors With Groutine Pool</>, value: 'multi-reactors-pool', },
          ]}>
          <TabItem value="multi-reactors">
            <div className={styles.topology}>
              <SVG src="/img/multi-reactors.svg" width="100%" />
            </div>
          </TabItem>
          <TabItem value="multi-reactors-pool">
            <div className={styles.topology}>
              <SVG src="/img/multi-reactors+thread-pool.svg" width="100%" />
            </div>
          </TabItem>
        </Tabs>
      </div>
    </section>
  )
}

function InstallationSection() {
  return (
    <section className={styles.installation}>
      <div className="container">
        <AnchoredH2 id="installation">Cross Platform</AnchoredH2>
        <div className="sub-title">Built on Linux, FreeBSD, DragonFly BSD, Darwin, Windows</div>

        <div className={styles.installationPlatforms}>
          <SVG src="/img/linux.svg" width="200px" />
          &nbsp;&nbsp;&nbsp;&nbsp;
          <SVG src="/img/apple.svg" width="200px" />
          &nbsp;&nbsp;&nbsp;&nbsp;
          <SVG src="/img/windows.svg" width="200px" />
        </div>

        <div className={styles.installationChecks}>
          <div>
            <i className="feather icon-package"></i> UNIX & Windows
          </div>
          <div>
            <i className="feather icon-cpu"></i> X86_64, ARM64
          </div>
          <div>
            <i className="feather icon-feather"></i> Light-weight
          </div>
          <div>
            <i className="feather icon-zap"></i> Ultra-fast
          </div>
        </div>

        <h3 className={styles.installSubTitle}>How to install gnet</h3>
        <div className="sub-title">
            <p className="hero--subsubtitle">`gnet` is available as a Go module
            and we highly recommend that you use `gnet` via <a href="https://go.dev/blog/using-go-modules" target="_blank">Go Modules</a>,
            with Go 1.11 Modules enabled (Go 1.11+), you can just simply add `import "github.com/panjf2000/gnet/v2"`
            to the codebase and run `go mod download/go mod tidy` or `go [build|run|test]` to download the necessary dependencies automatically.
            </p>
        </div>

        <p className="hero--subtitle">Run <strong><em>go get</em></strong> to download gnet:</p>

        <h4 className={styles.installSubTitle}>With v2</h4>
        <CodeBlock className="language-bash">go get -u github.com/panjf2000/gnet/v2</CodeBlock>

        <h4 className={styles.installSubTitle}>With v1</h4>
        <CodeBlock className="language-bash">go get -u github.com/panjf2000/gnet</CodeBlock>
      </div>
    </section>
  );
}

const hardware = `
# Hardware Environment
* 28 HT Cores Intel(R) Xeon(R) Gold 5120 CPU @ 3.20GHz
* 32GB RAM
* Dedicated Cisco 10-gigabit Ethernet switch
* Debian 12 "bookworm"
* Go1.19.x linux/amd64
`

function Performance() {
  return (
    <section className={styles.performance}>
      <div className="container">
        <AnchoredH2 id="performance">Performance</AnchoredH2>
        <div className="sub-title">Benchmarks on TechEmpower</div>
        <CodeBlock className="language-bash">{hardware}</CodeBlock>
        <br />
        <img src="/img/techempower-plaintext-top50-dark.jpg" alt="All languages" />
        <br />
        <p className="hero--subtitle">This is a leaderboard of the top <strong><em>50</em></strong> out of <strong><em>486</em></strong> frameworks that encompass various programming languages worldwide, in which <code>gnet</code> is ranked <strong><em>first</em></strong>.</p>
        <img src="/img/techempower-plaintext-topN-go-dark.png" alt="Go" />
        <br />
        <p className="hero--subtitle">This is the full framework ranking of Go and <code>gnet</code> tops all the other frameworks, which makes <code>gnet</code> the <strong><em>fastest</em></strong> networking framework in Go.</p>
        <br />
        <p className="hero--subtitle">To check the full ranking list, visit <Link to="https://www.techempower.com/benchmarks/#hw=ph&test=plaintext&section=data-r22">TechEmpower Benchmark <strong>Round 22</strong></Link>.</p>
        <p className="hero--subtitle">Note that the HTTP implementation of gnet on TechEmpower is half-baked and fine-tuned for benchmark purposes only and is far from production-ready.</p>
      </div>
    </section>
  );
}

function UseCases() {
  return (
    <section className={styles.usecases}>
      <div className="container">
        <AnchoredH2 id="usecases">Use cases</AnchoredH2>
        <div className="sub-title">The following companies/organizations use <code>gnet</code> as the underlying network service in production.</div>
        <table>
          <tbody>
            <tr>
              <td align="center" valign="middle">
                <a href="https://www.tencent.com/">
                  <img src="https://res.strikefreedom.top/static_res/logos/tencent_logo.png" width="200" />
                </a>
              </td>
              <td align="center" valign="middle">
                <a href="https://www.iqiyi.com/" target="_blank">
                  <img src="https://res.strikefreedom.top/static_res/logos/iqiyi-logo.png" width="200" />
                </a>
              </td>
              <td align="center" valign="middle">
                <a href="https://www.mi.com/global/" target="_blank">
                  <img src="https://res.strikefreedom.top/static_res/logos/mi-logo.png" width="200" />
                </a>
              </td>
              <td align="center" valign="middle">
                <a href="https://www.360.com/" target="_blank">
                  <img src="https://res.strikefreedom.top/static_res/logos/360-logo.png" width="200" />
                </a>
              </td>
            </tr>
            <tr>
              <td align="center" valign="middle">
                <a href="https://tieba.baidu.com/" target="_blank">
                  <img src="https://res.strikefreedom.top/static_res/logos/baidu-tieba-logo.png" width="200" />
                </a>
              </td>
              <td align="center" valign="middle">
                <a href="https://game.qq.com/" target="_blank">
                  <img src="https://res.strikefreedom.top/static_res/logos/tencent-games-logo.jpeg" width="200" />
                </a>
              </td>
              <td align="center" valign="middle">
                <a href="https://www.jd.com/" target="_blank">
                  <img src="https://res.strikefreedom.top/static_res/logos/jd-logo.png" width="200" />
                </a>
              </td>
              <td align="center" valign="middle">
                <a href="https://www.zuoyebang.com" target="_blank">
                  <img src="https://res.strikefreedom.top/static_res/logos/zuoyebang-logo.jpeg" width="200" />
                </a>
              </td>
            </tr>
            <tr>
              <td align="center" valign="middle">
                <a href="https://www.bytedance.com/" target="_blank">
                  <img src="https://res.strikefreedom.top/static_res/logos/ByteDance_Logo.png" width="200" />
                </a>
              </td>
            </tr>
          </tbody>
        </table>
        <br />
        <p className="hero--subtitle">If your projects are also using <code>gnet</code>, feel free to open a <Link to="https://github.com/panjf2000/gnet/pulls">pull request</Link> refreshing this list.</p>
      </div>
    </section>
  );
}

function Notice() {
  const newPost = fetchNewPost();
  const newHighlight = fetchNewHighlight();
  const items = [newPost, newHighlight];
  const item = _(items).compact().sortBy('date').value()[0];

  if (item) {
    return <Link to={item.permalink} className={styles.indexAnnouncement}>
      <span className="badge badge-primary">new</span>
      {item.title}
    </Link>
  } else {
    return null;
  }
}

function Home() {
  const context = useDocusaurusContext();
  const { siteConfig = {} } = context;

  // useEffect(() => {
  //   cloudify();
  // }, []);

  return (
    <Layout title={`${siteConfig.title} - ${siteConfig.tagline}`} description={siteConfig.tagline}>
      <header className={classnames('hero', 'hero--full-height', styles.indexHeroBanner)}>
        <div className="container container--fluid">
          <Notice />
          <h1>Build high-performance networking applications in Go</h1>
          <p className="hero--subtitle">
            On top of a variety of protocols of <Link to="https://github.com/gnet-io/gnet-examples">HTTP, RPC, WebSocket, Redis, etc.</Link>
          </p>
          <div className="hero--buttons">
            <Link to="https://github.com/panjf2000/gnet" className="button button--primary"><i className="feather icon-github"></i> View on Github</Link>
          </div>
          <SVG src="/img/diagram.svg" width="100%" />
          <p className="hero--subsubtitle">
            gnet is the <strong><em>fastest networking framework</em></strong> in Go.
          </p>
        </div>
      </header>
      <main>
        {features && features.length && <Features features={features} />}
        <Topologies />
        <InstallationSection />
        <Performance />
        <UseCases />
      </main>
    </Layout>
  );
}

export default Home;
