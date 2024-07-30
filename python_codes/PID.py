import matplotlib
from matplotlib import pyplot as plt

class PIDController:
    def __init__(self, Kp, Ki, Kd):
        self.Kp = Kp
        self.Ki = Ki
        self.Kd = Kd
        self.prev_error = 0
        self.integral = 0

    def compute(self, setpoint, current_value):
        error = setpoint - current_value
        self.integral += error
        derivative = error - self.prev_error
        control = self.Kp * error + self.Ki * self.integral + self.Kd * derivative
        next_value = current_value + control
        self.prev_error = error
        return next_value, error, control

# Example of PID controller instantiation
pid = PIDController(Kp=0.5, Ki=0.1, Kd=0.2)
current_value = 5
out = []
error = []
ctrl = []
for t in range(1,100):
    setpoint = t//4
    output, e, control = pid.compute(setpoint, current_value)
    current_value = output
    out.append(current_value)
    error.append(e)
    ctrl.append(control)
    # print(f"Control Output: {control_output}")

fig, axs = plt.subplots(3, 1)  # Corrected to plt.subplots
axs[0].plot(range(1,100),out)
axs[1].plot(range(1,100),error)
axs[2].plot(range(1,100),ctrl)
plt.show()