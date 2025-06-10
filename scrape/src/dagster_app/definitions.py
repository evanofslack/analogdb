import dagster as dg
from .resources import AnalogDBResource
from .assets import posts
from dotenv import load_dotenv

load_dotenv()

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
