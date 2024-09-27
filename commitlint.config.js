const Configuration= {
    extends: ['@commitlint/config-conventional'],
    formatter: '@commitlint/format',
    rules: {
        'body-empty': [1, 'never'],
        'body-case': [2, 'always', 'sentence-case'],
        'body-max-line-length': [1, 'always', 100],
        'references-empty': [1, 'never'],
        'signed-off-by': [2, 'always', 'Signed-off-by:'],
    },
};

module.exports = Configuration;
