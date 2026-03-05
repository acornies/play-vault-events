extends Control

@onready var animation: AnimationPlayer = $AnimationPlayer

# Called when the node enters the scene tree for the first time.
func _ready() -> void:
	pass # Replace with function body.


# Called every frame. 'delta' is the elapsed time since the previous frame.
func _process(delta: float) -> void:
	pass

func _on_vault_event() -> void:
	print("received data signal")
	animation.play("pulsate")
	
func pulsate() -> void:
	animation.play("pulsate")
