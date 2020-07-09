import React from 'react';

import BlogPostItem from '@theme/BlogPostItem';
import BlogListPaginator from '@theme/BlogListPaginator';
import CTA from '@site/src/components/CTA';
import Layout from '@theme/Layout';
import Link from '@docusaurus/Link';

import {enrichTags} from '@site/src/exports/tags';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import {viewedNewPost} from '@site/src/exports/newPost';

import './styles.css';

function BlogListPage(props) {
  const {metadata, items} = props;
  const context = useDocusaurusContext();
  const {siteConfig = {title: siteTitle}} = context;
  const {metadata: {post_tags: postTags}} = siteConfig.customFields;
  const enrichedTags = enrichTags(postTags, 'blog');
  const typeTags = enrichedTags.filter(tag => tag.category == 'type');
  const domainTags = enrichedTags.filter(tag => tag.category == 'domain');
  const isBlogOnlyMode = metadata.permalink === '/';
  const title = isBlogOnlyMode ? siteTitle : 'Blog';

  viewedNewPost();

  return (
    <Layout title={title} description="Gnet blog posts, articles, and tips from the Gnet core team.">
      <div className="blog-list container">
        <div className="blog-list--filters">
          <a href="/blog/rss.xml" style={{float: 'right', fontSize: '1.5em', marginTop: '0px', marginLeft: '-30px', width: '30px'}}><i className="feather icon-rss"></i></a>
          <h1>The Gnet Blog</h1>
          <p>Gnet is a high-performance, lightweight, non-blocking, event-driven networking framework written in pure Go, created by <Link to="https://github.com/panjf2000">Andy Pan</Link>.</p>

          <div className="margin-vert--lg">
            <CTA github={false} size="s" inline={true} style="highlight" />
          </div>

          <p className="margin-vert--sm">Looking for product updates &amp; announcements?</p>
          <p><Link to="/highlights/"><i className="feather icon-gift"></i> Check out the highlights section</Link></p>
        </div>
        <div className="blog-list--items">
          {items.map(({content: BlogPostContent}) => (
            <BlogPostItem
              key={BlogPostContent.metadata.permalink}
              frontMatter={BlogPostContent.frontMatter}
              metadata={BlogPostContent.metadata}
              truncated={BlogPostContent.metadata.truncated}>
              <BlogPostContent />
            </BlogPostItem>
          ))}
          <BlogListPaginator metadata={metadata} />
        </div>
      </div>
    </Layout>
  );
}

export default BlogListPage;
