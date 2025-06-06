import dagster as dg
from resources import AnalogDBResource
from assets import posts

defs = dg.Definitions(
    assets=[posts],
    resources={
        "analogdb": AnalogDBResource(
            base_url="https://api.analog.com",
            username=dg.EnvVar("ANALOGDB_USERNAME"),
            password=dg.EnvVar("ANALOGDB_PASSWORD"),
        )
    },
)

defs = dg.Definitions(assets=[posts])
