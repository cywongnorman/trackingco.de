const React = require('react')
const h = require('react-hyperscript')

const graphql = require('./graphql')

window.React = require('react')
window.ReactDOM = require('react-dom')

module.exports = React.createClass({
  componentDidMount () {
    let script = document.createElement('script')
    script.onload = () => { this.forceUpdate() }
    script.src = 'https://cdnjs.cloudflare.com/ajax/libs/graphiql/0.9.3/graphiql.min.js'
    document.body.appendChild(script)

    let link = document.createElement('link')
    link.rel = 'stylesheet'
    link.href = 'https://cdnjs.cloudflare.com/ajax/libs/graphiql/0.9.3/graphiql.min.css'
    document.head.appendChild(link)
  },

  render () {
    return (
      window.GraphiQL
      ? (
        h('.container', [
          h(window.GraphiQL, {
            fetcher: ({query, params}) => {
              console.log(arguments)
              graphql.query(query)
            }
          })
        ])
      )
      : h('div')
    )
  }
})
