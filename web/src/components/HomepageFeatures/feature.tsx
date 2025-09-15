import type {ComponentProps, ComponentType, ReactElement} from "react";
import clsx from "clsx";
import styles from "@site/src/components/HomepageFeatures/styles.module.css";
import Heading from "@theme/Heading";

type FeatureItem = {
    title: string;
    Svg: ComponentType<ComponentProps<'svg'>>;
    description: string;
};

function Feature({title, Svg, description}: FeatureItem): ReactElement<FeatureItem> {
    return (
        <div className={clsx('col col--4')}>
            <div className="text--center">
                <Svg className={styles.featureSvg} role="img"/>
            </div>
            <div className="text--center padding-horiz--md">
                <Heading as="h3">{title}</Heading>
                <p>{description}</p>
            </div>
        </div>
    );
}

export function FeatureList(): ReactElement<null> {
    return (
        <div className="row">
            <Feature
                title={'Backup your clusters'}
                description={"Securely backup your CloudNativePG clusters to object storage with configurable retention " +
                    "policies and compression options"}
                Svg={require('@site/static/img/undraw_going-up_g8av.svg').default}

            />
            <Feature
                title={'Restore to any point in time'}
                description={"Perform flexible restores to any point in time using a combination of " +
                    "base backups and WAL archives."}
                Svg={require('@site/static/img/undraw_season-change_ohe6.svg').default}
            />
            <Feature
                title={'Cloud-native architecture'}
                description={"Seamlessly integrate with all major cloud providers and on-premises object storage solutions."}
                Svg={require('@site/static/img/undraw_maintenance_rjtm.svg').default}
            />
        </div>
    )
}
