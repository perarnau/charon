import streamlit as st
import pandas as pd
import json
import numpy as np
from sh import tail


def process_stat(s):
    if "collectorId" in s:
        r = {
            "id": s["objectId"],
            "timestamp": s["objectTime"],
        }
        r.update(s["collectorStats"])
        return r

df = pd.DataFrame()
# st.line_chart(df)

if st.button('Say hello'):
    st.write('Why hello there')
else:
    st.write('Goodbye')

col1, col2, col3 = st.columns(3)

# def echo(x):
#     global df
#     new_collected = x['collectorStats']['nCollected']
#     df[time.now()] = new_collected
with col1, st.empty():
    for line in tail("-f", "/tmp/stat.txt", _iter=True):
        j = process_stat(json.loads(line))
        st.json(j)
        # print(j)
        # d = pd.DataFrame([j])
        # df = pd.concat([df, d])
        # st.line_chart(df, x="timestamp")

# c.subscribe('echo', echo)
# c.startMonitor()
# c.stopMonitor()
# c.unsubscribe('echo')

