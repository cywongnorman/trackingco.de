/** @format */

import React from 'react' // eslint-disable-line no-unused-vars

import SiteDetail from './SiteDetail'

export default function Main() {
  let domain = location.pathname.slice(1)

  return <SiteDetail domain={domain} />
}
