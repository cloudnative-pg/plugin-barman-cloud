import type {ReactElement} from 'react';
import styles from './styles.module.css';
import {FeatureList} from './feature';

export default function HomepageFeatures(): ReactElement<null> {
    return (
        <section className={styles.features}>
            <div className="container">
                <FeatureList/>
            </div>
        </section>
    );
}
