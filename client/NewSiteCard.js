const React = require('react')
const withClickOutside = require('react-click-outside')
const h = require('react-hyperscript')

const graphql = require('./graphql')

module.exports = withClickOutside(React.createClass({
  getInitialState () {
    return {
      name: '',
      writing: false
    }
  },

  q: `
    ($name: String!) {
      createSite(name: $name) {
        code
      }
    }
  `,

  mutate () {
    graphql.mutate(this.q, {name: this.state.name})
    .then(r => {
      this.props.onNewSiteCreated()
      this.setState(this.getInitialState())
    })
    .catch(console.log.bind(console))
  },

  render () {
    return h('.card.newsite', {id: this.props.code}, [
      h('.card-content', {onClick: this.startWriting}, [
        h('h4.title.is-5', this.state.writing ? 'give a name to the site' : 'add a new site'),
        this.state.writing
        ? h('.card-content', [
          h('form', {onSubmit: this.submit}, [
            h('input.input', {onChange: this.write, value: this.state.name}),
            h('button.button.is-primary', 'add')
          ])
        ])
        : h('i.fa.fa-plus')
      ])
    ])
  },

  startWriting () {
    this.setState({writing: true})
  },

  write (e) {
    this.setState({name: e.target.value})
  },

  submit (e) {
    e.preventDefault()
    this.mutate()
  },

  handleClickOutside () {
    this.setState({writing: false})
  }
}))
