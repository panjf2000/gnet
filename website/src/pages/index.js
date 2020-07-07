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
import { fetchNewPost } from '@site/src/exports/newPost';
import { fetchNewRelease } from '@site/src/exports/newRelease';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';

import _ from 'lodash';
import styles from './index.module.css';

import './index.css';

const AnchoredH2 = Heading('h2');

const features = [
  {
    title: 'Ultra Fast',
    icon: 'zap',
    description: (
      <>
        Built in <a href="https://www.golang.org/">Go</a>, Gnet is <a href="#performance">ultra fast and
        memory efficient</a>. It's designed to create a networking server framework for Go that performs on par with Redis and Haproxy for networking packets handling.
      </>
    ),
  },
  {
    title: 'Lock Free',
    icon: 'unlock',
    description: (
      <>
        Gnet is iock-free during the entire life cycle, which gets gnet out of synchronization.
      </>
    ),
  },
  {
    title: 'Concise APIs',
    icon: 'codepen',
    description: (
      <>
        Gnet provides concise APIs to users, it only exposes the essential APIs and takes over most of the tough work for users,
        minimizes the complexity of business code so that developers are able to concentrate on business logic instead of the underlying implementation.
      </>
    ),
  },
  {
    title: 'Multiple Porotocols',
    icon: 'shuffle',
    description: (
      <>
        Gnet supports multiple protocols/IPC mechanism: TCP, UDP and Unix Domain Socket, enabling you to develop
        a variety of networking applications.
      </>
    ),
  },
  {
    title: 'Cross Platform',
    icon: 'code',
    description: (
      <>
        Gnet is devised as a cross-platform framework, as a result, it works faultlessly on multiple platforms: Linux, FreeBSD, DragonFly BSD, Windows.
      </>
    ),
  },
  {
    title: 'Load Balancing',
    icon: 'shield',
    description: (
      <>
        Gnet supports multiple load-balancing algorithms: Round-Robin, Source Addr Hash and Least-Connections,
        users is able to choose the most suitable load-balancing algorithm based on their business scenario.
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
        <AnchoredH2 id="features">Why Gnet?</AnchoredH2>
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

const hardware = `
# Hardware
CPU: 28 HT Cores Intel(R) Xeon(R) Gold 5120 CPU @ 2.20GHz
Mem: 32GB RAM
OS : Ubuntu 18.04.3 4.15.0-88-generic #88-Ubuntu
Net: Switched 10-gigabit ethernet
Go : go1.14.x linux/amd64
`

function Performance() {
  return (
    <section className={styles.performance}>
      <div className="container">
        <AnchoredH2 id="performance">Performance</AnchoredH2>
        <div className="sub-title">Benchmarks on TechEmpower</div>
        <CodeBlock className="language-bash">{hardware}</CodeBlock>
        <br />
        <img src="/img/techempower-all.jpg" alt="All language" />
        <br />
        <p className="hero--subsubtitle">This is the top <strong><em>50</em></strong> on the framework ranking of all programming languages consists of a total of <strong><em>382 frameworks</em></strong> from all over the world.</p>

      </div>
    </section>
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
        <div className="sub-title">We highly recommend that you install gnet via Go Modules</div>
        
        <p className="hero--subtitle">Run <strong><em>go get</em></strong> to download gnet:</p>
        <CodeBlock className="language-bash">go get -u github.com/panjf2000/gnet</CodeBlock>
        <p className="hero--subsubtitle">gnet is available as a Go module, with Go 1.11 Modules support (Go 1.11+), just simply import "github.com/panjf2000/gnet" in your source code and go [build|run|test] will download the necessary dependencies automatically.</p>
      </div>
    </section>
  );
}

function Notice() {
  const newHighlight = fetchNewHighlight();
  const newPost = fetchNewPost();
  const newRelease = fetchNewRelease();
  const items = [newHighlight, newPost, newRelease];
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
  const { metadata: { latest_release } } = siteConfig.customFields;

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
            Gnet is the <strong><em>fastest networking framework</em></strong> in Go.
          </p>
        </div>
      </header>
      <main>
        {features && features.length && <Features features={features} />}
        <Topologies />
        <InstallationSection />
        <Performance />
      </main>
    </Layout>
  );
}

export default Home;
