import type {ReactNode} from 'react';
import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  Svg: React.ComponentType<React.ComponentProps<'svg'>>;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'Backup your clusters',
    Svg: require('@site/static/img/undraw_going-up_g8av.svg').default,
    description: (
      <>
        Securely backup your CloudNativePG clusters to object storage with
        configurable retention policies and compression options.
      </>
    ),
  },
  {
    title: 'Restore to any point in time',
    Svg: require('@site/static/img/undraw_season-change_ohe6.svg').default,
    description: (
      <>
        Perform flexible restores to any point in time using a combination of
        base backups and WAL archives.
      </>
    ),
  },
  {
    title: 'Cloud-native architecture',
    Svg: require('@site/static/img/undraw_maintenance_rjtm.svg').default,
    description: (
      <>
        Seamlessly integrate with all major cloud providers and on-premises object storage
        solutions.
      </>
    ),
  },
];

function Feature({title, Svg, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center">
        <Svg className={styles.featureSvg} role="img" />
      </div>
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
