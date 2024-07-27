import numpy as np
import matplotlib.pyplot as plt
# Constants
MOTOR_CAPACITY = 200
K_p = 1  # Proportional gain, needs tuning
total_active_capacity = 0


# Function to determine the number of motors needed based on the load and proportional control
def p_control_motors(current_load, motor_states):
    total_active_capacity = sum(MOTOR_CAPACITY for motor in range(motor_states))
    error = current_load - total_active_capacity
    control_signal = K_p * error
    motors_needed = (current_load + MOTOR_CAPACITY - 1) // MOTOR_CAPACITY
    return motors_needed, error, control_signal

# Example usage
control = []
setpoint = []
err = []
motors_needed = 0
change = []
for t in range(0,100):
    if t % 10 == 0:
        current_fpr = np.random.randint(0, 1201)  # Current load demand that varies randomly every 10 seconds between 0 to 1200
    motors_needed, error, control_signal = p_control_motors(current_fpr, motors_needed)
    # print("Motor needed:", motors_needed)
    setpoint.append(current_fpr)
    control.append(motors_needed)
    err.append(error//MOTOR_CAPACITY)

fig,axs = plt.subplots(3,1)
axs[0].plot(range(0,100), setpoint)
axs[1].plot(range(0,100), control)
axs[2].plot(range(0,100), err)

plt.show()
